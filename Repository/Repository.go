package Repository

import (
	"database/sql"
	"errors"
	"fmt"
	mssql "github.com/denisenkom/go-mssqldb"
	"github.com/gin-gonic/gin"
	"httpRequestName/Core"
	"httpRequestName/DB"
	"httpRequestName/Model"
	"reflect"
	"sort"
	"strings"
)

// Repository is the generic repository struct for handling database operations.
type Repository[T any] struct {
}

// NewRepository creates a new instance of the Repository.
func NewRepository[T any]() *Repository[T] {
	return &Repository[T]{}
}

// Insert inserts a new record for the provided entity into the database and returns success status and error, if any.
func (repo *Repository[T]) Insert(entity T, c *gin.Context) (bool, error) {
	// Open DB connection
	db, ctx := DB.SqlOpen(c)
	defer db.Close()

	// 1. GetAll field-value map from entity
	fieldMap := getFieldMap(entity) // map[string]interface{}
	if len(fieldMap) == 0 {
		return false, fmt.Errorf("entity has no fields to insert")
	}

	// 2. Build dynamic SQL
	columns := ""
	values := ""
	args := []interface{}{}
	i := 1
	for col, val := range fieldMap {
		if i > 1 {
			columns += ", "
			values += ", "
		}
		columns += col
		values += fmt.Sprintf("@p%d", i)
		args = append(args, sql.Named(fmt.Sprintf("p%d", i), val))
		i++
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", getTableName[T](), columns, values)

	// 3. Prepare and execute the insert statement
	stmt, err := db.PrepareContext(*ctx, query)
	if err != nil {
		return false, fmt.Errorf("statement preparation error: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(*ctx, args...)
	if err != nil {
		//return false, fmt.Errorf("statement execution error: %w", err)
		return false, fmt.Errorf("exec error: %w | Query: %s | Args: %+v", err, query, args)

	}

	return true, nil
}

func (repo *Repository[T]) BulkInsert(entities []T, c *gin.Context, typeName string, procName string, size ...int) error {
	batchSize := 1000
	if len(size) > 0 && size[0] > 0 {
		batchSize = size[0]
	}
	if len(entities) == 0 {
		return fmt.Errorf("no records to insert")
	}

	db, ctx := DB.SqlOpen(c)
	defer db.Close()

	tx, err := db.BeginTx(*ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	//const batchSize = 1000 // MSSQL TVP optimal size
	for i := 0; i < len(entities); i += batchSize {
		end := i + batchSize
		if end > len(entities) {
			end = len(entities)
		}
		batch := entities[i:end]

		tvp := mssql.TVP{
			TypeName: typeName, // e.g. "dbo.UserTableType"
			Value:    batch,
		}

		_, err := tx.ExecContext(*ctx, fmt.Sprintf("EXEC %s @Data", procName), sql.Named("Data", tvp))
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute TVP proc: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (repo *Repository[T]) BulkInsertBatched(entities []T, c *gin.Context, batchSize int) (bool, error) {
	if len(entities) == 0 {
		return false, fmt.Errorf("no records to insert")
	}

	db, ctx := DB.SqlOpen(c)
	defer db.Close()

	tx, err := db.BeginTx(*ctx, nil)
	if err != nil {
		return false, fmt.Errorf("failed to begin transaction: %w", err)
	}

	paramCounter := 1
	tableName := getTableName[T]()
	allFields := getAllColumnNames(entities[0])

	for i := 0; i < len(entities); i += batchSize {
		end := i + batchSize
		if end > len(entities) {
			end = len(entities)
		}
		batch := entities[i:end]

		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES ", tableName, strings.Join(allFields, ", "))
		var placeholders []string
		var args []interface{}

		for _, entity := range batch {
			fieldMap := getFieldMap(entity)
			var valuePlaceholders []string
			for _, col := range allFields {
				ph := fmt.Sprintf("@p%d", paramCounter)
				valuePlaceholders = append(valuePlaceholders, ph)
				args = append(args, sql.Named(fmt.Sprintf("p%d", paramCounter), fieldMap[col]))
				paramCounter++
			}
			placeholders = append(placeholders, fmt.Sprintf("(%s)", strings.Join(valuePlaceholders, ", ")))
		}

		query += strings.Join(placeholders, ", ")

		_, err := tx.ExecContext(*ctx, query, args...)
		if err != nil {
			tx.Rollback()
			return false, fmt.Errorf("batch insert failed: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return false, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return true, nil
}

// BulkUpdate updates multiple fields for all records matching the filter.
func (repo *Repository[T]) BulkUpdateWithFilter(
	updates map[string]interface{},
	filter map[string]interface{},
	c *gin.Context,
) (bool, error) {
	// 1. DB bağlantısını aç
	db, ctx := DB.SqlOpen(c)
	defer db.Close()

	// 2. Eğer değişiklik yoksa, çık
	if len(updates) == 0 {
		return true, nil
	}

	// 3. UPDATE sorgusunu oluştur
	query := "UPDATE " + getTableName[T]() + " SET "
	var args []interface{}
	paramIndex := 1

	// 3a. SET kısmı
	i := 0
	for col, val := range updates {
		if i > 0 {
			query += ", "
		}

		if raw, ok := val.(Model.RawValue); ok {
			// Use raw SQL expression directly
			query += fmt.Sprintf("%s = %s", col, raw.Expr)
		} else {
			paramName := fmt.Sprintf("p%d", paramIndex)
			query += fmt.Sprintf("%s = @%s", col, paramName)
			args = append(args, sql.Named(paramName, val))
			paramIndex++
		}
		i++
	}
	// 3b. WHERE kısmı
	if len(filter) > 0 {
		query += " WHERE "
		j := 0
		for col, val := range filter {
			if j > 0 {
				query += " AND "
			}
			paramName := fmt.Sprintf("p%d", paramIndex)
			query += fmt.Sprintf("%s = @%s", col, paramName)
			args = append(args, sql.Named(paramName, val))
			paramIndex++
			j++
		}
	}

	// 4. Sorguyu hazırla ve çalıştır
	stmt, err := db.PrepareContext(*ctx, query)
	if err != nil {
		return false, fmt.Errorf("Statement preparation error: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(*ctx, args...)
	if err != nil {
		return false, fmt.Errorf("Statement execution error: %w", err)
	}

	return true, nil
}

// Update method updates the fields of a record in the database.
func (repo *Repository[T]) Update(oldEntity T, newEntity T, identifier string, c *gin.Context) (bool, error) {
	// Open DB connection
	db, ctx := DB.SqlOpen(c)
	defer db.Close()

	// 1. GetAll the changed fields between old and new entity
	changes := Core.GetChangedFields(oldEntity, newEntity)
	if len(changes) == 0 {
		// No changes detected, no need to update
		return true, nil
	}

	// 2. Build dynamic SQL for update
	query := "UPDATE " + getTableName[T]() + " SET "
	args := []interface{}{}
	i := 1
	for col, val := range changes {
		if i > 1 {
			query += ", "
		}
		query += fmt.Sprintf("%s = @p%d", col, i)
		args = append(args, sql.Named(fmt.Sprintf("p%d", i), val))
		i++
	}
	query += " WHERE UserName = @p" + fmt.Sprintf("%d", i)
	args = append(args, sql.Named(fmt.Sprintf("p%d", i), identifier))

	// 3. Prepare and execute the update statement
	stmt, err := db.PrepareContext(*ctx, query)
	if err != nil {
		return false, fmt.Errorf("Statement preparation error: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(*ctx, args...)
	if err != nil {
		return false, fmt.Errorf("Statement execution error: %w", err)
	}

	return true, nil
}

// BatchUpdate updates multiple records in the database based on specified old→new pairs.
// – Her farklı SET ifadesi için sorgu ön hazırlayıp cache’ler.
// – Kolon isimlerini sıralayarak deterministik parametre sırası sağlar.
// – Tek bir transaction içinde, hatada rollback ile çalışır.
func (repo *Repository[T]) BatchUpdate(
	pairs []struct{ Old, New T },
	c *gin.Context,
	identifierField string, // örn: "UserName"
	batchSize int,
) error {
	// Eğer batchSize geçersizse tüm liste tek seferde işlenir
	if batchSize <= 0 {
		batchSize = len(pairs)
	}

	db, ctx := DB.SqlOpen(c)
	defer db.Close()

	// Tek transaction açılıyor; herhangi bir hata tüm batch’leri geri alır
	tx, err := db.BeginTx(*ctx, nil)
	if err != nil {
		return fmt.Errorf("transaction start error: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Sorgu hazırlama cache’i
	stmtCache := make(map[string]*sql.Stmt)

	// pairs’ı batchSize kadar parçala
	for start := 0; start < len(pairs); start += batchSize {
		end := start + batchSize
		if end > len(pairs) {
			end = len(pairs)
		}
		chunk := pairs[start:end]

		for _, pair := range chunk {
			// Değişen alanları bul
			changes := Core.GetChangedFields(pair.Old, pair.New)
			if len(changes) == 0 {
				continue
			}

			// Kolon isimlerini deterministik sıraya al
			cols := make([]string, 0, len(changes))
			for col := range changes {
				cols = append(cols, col)
			}
			sort.Strings(cols)

			// SET ifadelerini hazırla
			setParts := make([]string, len(cols))
			for i, col := range cols {
				setParts[i] = fmt.Sprintf("%s = @p%d", col, i+1)
			}

			// Tam sorguyu oluştur
			query := fmt.Sprintf(
				"UPDATE %s SET %s WHERE %s = @p%d",
				getTableName[T](),
				strings.Join(setParts, ", "),
				identifierField,
				len(cols)+1,
			)

			// PrepareContext ya da cache’den al
			stmt, ok := stmtCache[query]
			if !ok {
				stmt, err = tx.PrepareContext(*ctx, query)
				if err != nil {
					tx.Rollback()
					return fmt.Errorf("prepare error: %w | query: %s", err, query)
				}
				stmtCache[query] = stmt
			}

			// Parametreleri sırayla ekle
			args := make([]interface{}, 0, len(cols)+1)
			for i, col := range cols {
				args = append(args, sql.Named(fmt.Sprintf("p%d", i+1), changes[col]))
			}

			// Identifier değerini al
			v := reflect.ValueOf(pair.New)
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}
			field := v.FieldByName(identifierField)
			if !field.IsValid() {
				tx.Rollback()
				return fmt.Errorf("invalid identifier field '%s'", identifierField)
			}
			args = append(args, sql.Named(
				fmt.Sprintf("p%d", len(cols)+1),
				field.Interface(),
			))

			// Sorguyu çalıştır
			if _, err = stmt.ExecContext(*ctx, args...); err != nil {
				tx.Rollback()
				return fmt.Errorf("execution error: %w | query: %s", err, query)
			}
		}
	}

	// Hazırlanan tüm statement’ları kapat
	for _, st := range stmtCache {
		st.Close()
	}

	// Commit tüm transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit error: %w", err)
	}

	return nil
}

// getTableName returns the table name for a given type T (this may need to be customized).
func getTableName[T any]() string {
	// Use reflection to check if the struct has a TableName method
	// If it does, call it to get the table name
	var tableName string
	if v, ok := any(new(T)).(interface {
		TableName() string
	}); ok {
		tableName = v.TableName()
	} else {
		// Fallback: Use the struct name as the table name
		tableName = strings.ToUpper(reflect.TypeOf(new(T)).Elem().Name())
	}

	if strings.Contains(tableName, "; \t\n") {
		panic("Invalid table name: " + tableName)
	}
	return tableName
}

// GetAll retrieves all records from the database for a given model
func (repo *Repository[T]) GetAll(c *gin.Context) ([]T, error) {
	records := make([]T, 0)

	db, ctx := DB.SqlOpen(c)
	defer db.Close()

	query := "SELECT * FROM " + getTableName[T]() + " WHERE ISNULL(IsDeleted, 0) = 0"
	rows, err := db.QueryContext(*ctx, query)
	if err != nil {
		fmt.Println("Query error:", err)
		return records, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		fmt.Println("Error getting columns:", err)
		return records, err
	}
	fmt.Println("Columns retrieved:", columns)

	for rows.Next() {
		var record T
		columnPointers := make([]interface{}, len(columns))

		for i, col := range columns {
			field := getStructFieldByName(&record, col)
			if field.IsValid() && field.CanAddr() {
				columnPointers[i] = field.Addr().Interface()
			} else {
				var dummy interface{}
				columnPointers[i] = &dummy
			}
		}

		err := rows.Scan(columnPointers...)
		if err != nil {
			fmt.Println("Scan error:", err)
			return records, err
		}

		records = append(records, record)
	}

	if err = rows.Err(); err != nil {
		fmt.Println("Rows iteration error:", err)
		return records, err
	}

	if len(records) == 0 {
		fmt.Println("No records found")
		return records, errors.New("persons not found")
	}

	return records, nil
}

func (repo *Repository[T]) GetByID(c *gin.Context, id int) (*T, error) {
	var record *T = new(T) // record'u pointer olarak tanımlıyoruz
	db, ctx := DB.SqlOpen(c)
	defer db.Close()

	query := "SELECT * FROM " + getTableName[T]() + " WHERE ISNULL(IsDeleted, 0) = 0 AND Id = @p1"
	rows, err := db.QueryContext(*ctx, query, id) // Tek satır bekliyoruz ama rows dönecek
	if err != nil {
		fmt.Println("Query error:", err)
		return record, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		fmt.Println("Error getting columns:", err)
		return record, err
	}

	if rows.Next() {
		columnPointers := make([]interface{}, len(columns))

		// Pointer'lar ile sütunları eşleştiriyoruz
		for i, col := range columns {
			field := getStructFieldByName(record, col) // artık record bir pointer
			if field.IsValid() && field.CanAddr() {
				columnPointers[i] = field.Addr().Interface()
			} else {
				var dummy interface{}
				columnPointers[i] = &dummy
			}
		}

		err = rows.Scan(columnPointers...)
		if err != nil {
			fmt.Println("Scan error:", err)
			return record, err
		}

		return record, nil
	}
	return record, nil // Eğer sonuç yoksa boş bir pointer döndürüyoruz
}

// getStructFieldByName returns the struct field that matches the column name.
func getStructFieldByName(record interface{}, columnName string) reflect.Value {
	v := reflect.ValueOf(record).Elem()
	t := v.Type()

	columnName = strings.TrimSpace(columnName)
	columnNameLower := strings.ToLower(columnName)

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		structField := t.Field(i)

		// Field Name kontrolü
		if strings.ToLower(structField.Name) == columnNameLower {
			return field
		}

		// JSON tag kontrolü
		jsonTag := structField.Tag.Get("json")
		if jsonTag != "" {
			jsonTagParts := strings.Split(jsonTag, ",")
			jsonTagName := jsonTagParts[0]

			if strings.ToLower(jsonTagName) == columnNameLower {
				return field
			}
		}
	}

	return reflect.Value{}
}

func getFieldMap[T any](record T) map[string]interface{} {
	result := make(map[string]interface{})

	v := reflect.ValueOf(record)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return result
		}
		v = v.Elem()
	}
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		structField := t.Field(i)

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		jsonTag := structField.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}
		columnName := structField.Name
		if jsonTag != "" {
			tagParts := strings.Split(jsonTag, ",")
			if tagParts[0] != "" && tagParts[0] != "-" {
				columnName = tagParts[0]
			}
		}

		result[columnName] = field.Interface()
	}

	return result
}

func getAllColumnNames[T any](entity T) []string {
	columns := []string{}
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		structField := t.Field(i)

		// Skip unexported fields
		if !v.Field(i).CanInterface() {
			continue
		}

		jsonTag := structField.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		columnName := structField.Name
		if jsonTag != "" {
			tagParts := strings.Split(jsonTag, ",")
			if tagParts[0] != "" && tagParts[0] != "-" {
				columnName = tagParts[0]
			}
		}
		columns = append(columns, columnName)
	}
	return columns
}

package Repository

import (
	"database/sql"
	"errors"
	"fmt"
	"httpRequestName/Core"
	"httpRequestName/DB"
	"httpRequestName/Model"
	"reflect"
	"sort"
	"strings"
	"sync"

	mssql "github.com/denisenkom/go-mssqldb"
	"github.com/gin-gonic/gin"
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

	//if strings.Contains(tableName, "; \t\n") {
	if strings.ContainsAny(tableName, ";\t \n") {
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

// GetAllWithPaging
// Paging + sıralama [ASC/DESC] parametrik sekilde kayıtları getirir. Default "ASC"
//
// Uyari:
// 1- orderBy SADECE "Id" veya "CreatedDate" olabilir. Istenirse degistirilebilir
// 2- Başka bir değer gelirse veya bos ise default "Id" kullanılır.
// 3- ASC/DESC yönü bool ile belirlenir. Default degeri "ASC"'dir
//
// Hata kontrolleri:
//   - orderBy = CreatedDate seçilmişse ama tabloda CreatedDate yoksa,
//     hatayı yakalayıp ORDER BY Id ile tekrar dener. Fazladan ekledim. Cikarilabilir.

// Kullanım Ornekleri:
//
//	repo.GetAllWithPaging(c, 1, 10)                 // Id ASC (default)
//	repo.GetAllWithPaging(c, 1, 10, "Id", false)     // Id DESC
//	repo.GetAllWithPaging(c, 1, 10, "Id")            // Id ASC
//	repo.GetAllWithPaging(c, 1, 10, "CreatedDate")   // CreatedDate ASC
//	repo.GetAllWithPaging(c, 1, 10, true)            // Id ASC
//	repo.GetAllWithPaging(c, 1, 10, false)           // Id DESC
func (repo *Repository[T]) GetAllWithPaging(
	c *gin.Context,
	page int,
	pageSize int,
	params ...interface{},
) ([]T, error) {

	records := make([]T, 0)

	// --- Varsayılanlar ---
	requestedOrderBy := ""
	orderDirection := "ASC"

	// --- Variadic parametreleri çözümle (string=orderBy, bool=isAscending) ---
	for _, p := range params {
		switch v := p.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				requestedOrderBy = strings.TrimSpace(v)
			}
		case bool:
			if v {
				orderDirection = "ASC"
			} else {
				orderDirection = "DESC"
			}
		}
	}

	// --- orderBy whitelist (sadece Id veya CreatedDate) ---
	orderBy := "Id" // default
	// Her zaman kucuk harfe cevirildi..
	switch strings.ToLower(requestedOrderBy) {
	case "id":
		orderBy = "Id"
	case "createddate":
		orderBy = "CreatedDate"
	}

	// --- Paging kontrolleri ---
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	db, ctx := DB.SqlOpen(c)
	defer db.Close()

	tableName := getTableName[T]()

	// Sorguyu çalıştıran küçük bir fonksiyon (tekrar denemek için)
	runQuery := func(ob string) ([]T, error) {
		out := make([]T, 0)

		query := fmt.Sprintf(`
			SELECT *
			FROM %s
			WHERE ISNULL(IsDeleted, 0) = 0
			ORDER BY %s %s
			OFFSET @offset ROWS
			FETCH NEXT @pageSize ROWS ONLY
		`, tableName, ob, orderDirection)

		rows, err := db.QueryContext(
			*ctx,
			query,
			sql.Named("offset", offset),
			sql.Named("pageSize", pageSize),
		)
		if err != nil {
			return out, err
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			return out, err
		}

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

			if err := rows.Scan(columnPointers...); err != nil {
				return out, err
			}

			out = append(out, record)
		}

		if err := rows.Err(); err != nil {
			return out, err
		}

		if len(out) == 0 {
			return out, errors.New("kayıt bulunamadı")
		}

		return out, nil
	}

	// 1) İlk deneme: seçilen orderBy ile
	result, err := runQuery(orderBy)
	if err == nil {
		return result, nil
	}

	// 2) Eğer CreatedDate seçili ve kolon yok hatası geldiyse -> Id ile tekrar dene
	if strings.EqualFold(orderBy, "CreatedDate") && isInvalidColumnNameErr(err) {
		return runQuery("Id")
	}

	return records, err
}

// isInvalidColumnNameErr
// SQL Server'da kolon adı geçersiz olduğunda gelen hatayı yakalamaya çalışır.
// (Genellikle: "Invalid column name 'CreatedDate'.")
func isInvalidColumnNameErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "invalid column name") || strings.Contains(msg, "geçersiz sütun adı")
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

type Op int

const (
	OpEq   Op = iota // =
	OpNe             // !=
	OpGt             // >
	OpGte            // >=
	OpLt             // <
	OpLte            // <=
	OpLike           // LIKE
)

func (o Op) SQL() (string, error) {
	switch o {
	case OpEq:
		return "=", nil
	case OpNe:
		return "!=", nil
	case OpGt:
		return ">", nil
	case OpGte:
		return ">=", nil
	case OpLt:
		return "<", nil
	case OpLte:
		return "<=", nil
	case OpLike:
		return "LIKE", nil
	default:
		return "", fmt.Errorf("desteklenmeyen operator")
	}
}

type Filter struct {
	Field string
	Op    Op
	Value interface{}
}

// FindOne
// Filtreye uyan TEK kaydı döner (SQL Server TOP 1)
//
// Davranış:
// - orderBy verilmezse -> Default => Id
// - asc verilmezse     -> Default => ASC

// Kullanım:
//
//	repo.FindOne(c, Repository.Filter{Field:"UserName", Op:OpEq, Value:"ali"})
//	repo.FindOne(c, Repository.Filter{Field:"Age", Op:OpGt, Value:18}, false) // DESC
//  user, err := repo.FindOne(c, Repository.Filter{Field: "Id", Op: Repository.OpEq, Value: id})
//  user, err := repo.FindOne(c, Repository.Filter{Field: "Id", Op: Repository.OpGt, Value: id})

// encUser, err := Core.Encrypt("borsoft", shared.Config.SECRETKEY)
// user, err := repo.FindOne(c, Repository.Filter{Field: "UserName", Op: Repository.OpEq, Value: encUser})

/*
OR Example

repo.FindOne(c,
Repository.Filter{Field:"IsActive", Op:OpEq, Value:true},
Or(
Where(Repository.Filter{Field:"Role", Op:OpEq, Value:"Admin"}),
Where(Repository.Filter{Field:"Age", Op:OpGt, Value:30}),
),
)
*/
func (repo *Repository[T]) FindOne(
	c *gin.Context,
	params ...interface{},
) (*T, error) {

	// Varsayılanlar. "Id" Default deger...
	orderBy := "Id"
	asc := true

	var rootExpr Expr = nil
	pending := make([]Expr, 0, 4) // düz Filter'lar burada birikir (AND)

	for _, p := range params {
		switch v := p.(type) {

		case Filter:
			//Default Filter'lar otomatik AND ile baglandi..
			pending = append(pending, Pred{F: v})

		case Expr:
			// önce Pending Filter'ları root'a AND grubu olarak bağlanir..
			if len(pending) > 0 {
				andGroup := Group{Op: LogicAnd, Items: pending}
				if rootExpr == nil {
					rootExpr = andGroup
				} else {
					rootExpr = Group{Op: LogicAnd, Items: []Expr{rootExpr, andGroup}}
				}
				pending = nil
			}

			// Expr'i Root'a eklemir (Default bağlama kosulu => AND)
			if rootExpr == nil {
				rootExpr = v
			} else {
				rootExpr = Group{Op: LogicAnd, Items: []Expr{rootExpr, v}}
			}

		case string:
			if strings.TrimSpace(v) != "" {
				orderBy = strings.TrimSpace(v)
			}

		case bool:
			asc = v

		default:
			return nil, fmt.Errorf("desteklenmeyen parametre tipi: %T", p)
		}
	}

	// döngü bitti, Pending durumu var ise root'a eklenir..
	if len(pending) > 0 {
		andGroup := Group{Op: LogicAnd, Items: pending}
		if rootExpr == nil {
			rootExpr = andGroup
		} else {
			rootExpr = Group{Op: LogicAnd, Items: []Expr{rootExpr, andGroup}}
		}
	}
	//(orderBy normalize / whitelist) OrderBy sadece Id veya CreatedDate'e gore yapilir..
	requestedOrderBy := strings.TrimSpace(orderBy)
	switch strings.ToLower(requestedOrderBy) {
	case "":
		// boşsa default kalsın: Id
	case "id":
		orderBy = "Id"
	case "createddate":
		orderBy = "CreatedDate"
	default:
		// bilinmeyen gelirse güvenli fallback
		orderBy = "Id"
	}

	// --------------------
	// orderBy doğrulama (struct alanları whitelist)
	//!!!YUKARIDAKI "CreatedDate" ve "Id" alanlari cikarilirsa bu kisim [allowed] elzem olur. Injection Guvenlik amacli. Yukarisi yeterli ise bu kisim [allowed] performans amacli cikarilabilir.
	// --------------------
	allowed := buildAllowedFieldSetCached[T]()

	//Sadece whitelist'deki alanlara izin verilir. Performans amacli Cacheleniyor.'
	if !allowed[strings.ToLower(orderBy)] {
		return nil, fmt.Errorf("izin verilmeyen orderBy alanı: %s", orderBy)
	}

	// --------------------
	// WHERE oluştur
	// --------------------
	//whereSQL, args, err := buildWhereFromFilters[T](filters, allowed)
	whereSQL, args, err := buildWhereFromExpr[T](rootExpr, allowed)
	if err != nil {
		return nil, err
	}

	// Soft delete
	if strings.TrimSpace(whereSQL) == "" {
		whereSQL = " WHERE ISNULL(IsDeleted, 0) = 0"
	} else {
		whereSQL += " AND ISNULL(IsDeleted, 0) = 0"
	}

	dir := "ASC"
	if !asc {
		dir = "DESC"
	}

	// --------------------
	// SQL (TOP 1)
	// --------------------
	db, ctx := DB.SqlOpen(c)
	defer db.Close()

	query := fmt.Sprintf(`
		SELECT TOP (1) *
		FROM %s
		%s
		ORDER BY %s %s
	`, getTableName[T](), whereSQL, orderBy, dir)

	rows, err := db.QueryContext(*ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	if !rows.Next() {
		return nil, errors.New("kayıt bulunamadı")
	}

	var record T
	ptrs := make([]interface{}, len(cols))
	for i, col := range cols {
		f := getStructFieldByName(&record, col)
		if f.IsValid() && f.CanAddr() {
			ptrs[i] = f.Addr().Interface()
		} else {
			var dummy interface{}
			ptrs[i] = &dummy
		}
	}

	if err := rows.Scan(ptrs...); err != nil {
		return nil, err
	}

	return &record, nil
}

// buildAllowedFieldSet
// Model T içindeki alanları (FieldName + json tag) whitelist olarak çıkarır
// Injection Engelleme amacli => Field: "Id; DROP TABLE Users" gibi...
// Performans amacli Cacheliyorum..
var allowedFieldCache sync.Map // key: reflect.Type, value: map[string]bool
func buildAllowedFieldSetCached[T any]() map[string]bool {
	var sample T
	t := reflect.TypeOf(sample)
	if t == nil {
		return map[string]bool{}
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if val, ok := allowedFieldCache.Load(t); ok {
		return val.(map[string]bool)
	}

	allowed := make(map[string]bool)
	if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumField(); i++ {
			sf := t.Field(i)
			if sf.PkgPath != "" { // unexported
				continue
			}
			allowed[strings.ToLower(sf.Name)] = true

			jsonTag := sf.Tag.Get("json")
			if jsonTag != "" && jsonTag != "-" {
				tagName := strings.Split(jsonTag, ",")[0]
				if tagName != "" {
					allowed[strings.ToLower(tagName)] = true
				}
			}
		}
	}

	allowedFieldCache.Store(t, allowed)
	return allowed
}

// page/pageSize verilmezse: tüm kayıtları döndürür (paging yok).
// Paging varsa: ORDER BY zorunlu olduğu için default "Id ASC" kullanılır.
// orderBy sadece "Id" veya "CreatedDate" olabilir (aksi halde "Id").

// Parametreler (variadic):
//   Repository.Filter  => WHERE koşulu
//   int (1. int)       => page
//   int (2. int)       => pageSize
//   string             => orderBy ("Id" | "CreatedDate")
//   bool               => ascending (true=ASC, false=DESC)

// Soyle Cagrilabilir:
//
//	repo.FindMany(c) // hepsi
//	repo.FindMany(c, Repository.Filter{Field:"Age", Op:Repository.OpGte, Value:18}) // WHERE Age >= 18
//	repo.FindMany(c, 1, 50) // page=1, pageSize=50 (Id ASC) Paging Ilk 50 kayit..
//	repo.FindMany(c, 2, 10, "CreatedDate", false, Repository.Filter{Field:"UserName", Op:Repository.OpLike, Value:"%ali%"}) // page=2, pageSize=10 (CreatedDate DESC), WHERE UserName LIKE "%ali%"

/*
AND Example

repo.FindMany(c,

		Repository.Filter{Field: "Age", Op: Repository.OpGte, Value: 15},
		Repository.Filter{Field: "Gender", Op: Repository.OpEq, Value: "Male"},
	)

-----------------------------
OR Example

expr := Or(

	Where(Repository.Filter{Field:"UserName", Op:OpEq, Value:"ali"}),
	Where(Repository.Filter{Field:"UserName", Op:OpEq, Value:"veli"}),

)
repo.FindMany(c, expr)

-----------------------------
OR, ORDERBY DESC Example

repo.FindMany(

		c,
		"CreatedDate",
		false,
		Repository.Filter{Field: "Age", Op: Repository.OpLte, Value: 50},
		Repository.Or(
			Repository.Where(Repository.Filter{Field: "Name", Op: Repository.OpEq, Value: "Bora"}),
			Repository.Where(Repository.Filter{Field: "Name", Op: Repository.OpEq, Value: "Secil"}),
		),
	) // OrderBy(CreatedDate, DESC) WHERE Age <= 50 AND (Name = "Bora" OR Name = "Secil")
*/
func (repo *Repository[T]) FindMany(
	c *gin.Context,
	params ...interface{},
) ([]T, error) {

	out := make([]T, 0)

	// ---- Varsayılanlar ----
	page := 0
	pageSize := 0
	asc := true
	requestedOrderBy := ""

	intCount := 0

	var rootExpr Expr = nil
	pending := make([]Expr, 0, 4)

	// Not: FindMany’de zaten page/pageSize/orderBy/asc parsing’in vardıysa,
	// asc burada set ediliyorsa aynı asc değişkenini kullan.
	for _, p := range params {
		switch v := p.(type) {

		case Filter:
			pending = append(pending, Pred{F: v})

		case Expr:
			if len(pending) > 0 {
				andGroup := Group{Op: LogicAnd, Items: pending}
				if rootExpr == nil {
					rootExpr = andGroup
				} else {
					rootExpr = Group{Op: LogicAnd, Items: []Expr{rootExpr, andGroup}}
				}
				pending = nil
			}

			if rootExpr == nil {
				rootExpr = v
			} else {
				rootExpr = Group{Op: LogicAnd, Items: []Expr{rootExpr, v}}
			}

			//OrderBy Field parametresi Id - CreatedDate
		case string:
			if strings.TrimSpace(v) != "" {
				requestedOrderBy = strings.TrimSpace(v)
			}

		case int:
			intCount++
			if intCount == 1 {
				page = v
			} else if intCount == 2 {
				pageSize = v
			}

		// FindMany bool asc or desc parametresi
		case bool:
			asc = v

		// FindMany’de paging/orderBy parametrelerini parse ediyorsan onlar da burada duracak
		// case int: ...
		// case string: ...

		default:
			return nil, fmt.Errorf("desteklenmeyen parametre tipi: %T", p)
		}
	}

	if len(pending) > 0 {
		andGroup := Group{Op: LogicAnd, Items: pending}
		if rootExpr == nil {
			rootExpr = andGroup
		} else {
			rootExpr = Group{Op: LogicAnd, Items: []Expr{rootExpr, andGroup}}
		}
	}

	// PAGING
	// hiç int yoksa -> paging yok (hepsi)
	// sadece 1 int geldiyse -> bu pageSize yapilir. Default page=1 olur.. (kolay kullanım)
	// Tek int geldiyse: FindMany(c, 50) => pageSize=50, page=1
	if intCount == 1 {
		pageSize = page
		page = 1
	}

	// Negatif pageSize paging'i kapatir
	if pageSize < 0 {
		pageSize = 0
	}

	// Paging açıksa page en az 1 olmalı
	if pageSize > 0 && page < 1 {
		page = 1
	}

	// Max pageSize 100 seklinde sinirlandirlabilir..
	/*if pageSize > 100 {
		pageSize = 100
	}*/

	// orderBy whitelist (Id/CreatedDate) Sade 2 alan guvenlik amacli degisitirilebilir. Default: Paging var ise => Id, ASC default
	orderBy := "Id" // default her zaman Id
	switch strings.ToLower(requestedOrderBy) {
	case "":
		// boşsa default kalsın: Id
	case "id":
		orderBy = "Id"
	case "createddate":
		orderBy = "CreatedDate"
	default:
		// bilinmeyen gelirse güvenli fallback
		orderBy = "Id"
	}

	orderDir := "ASC"
	if !asc {
		orderDir = "DESC"
	}

	// WHERE generate (Performans amacli Field whitelist cached)
	allowed := buildAllowedFieldSetCached[T]()
	//whereSQL, args, err := buildWhereFromFilters[T](filters, allowed)
	whereSQL, args, err := buildWhereFromExpr[T](rootExpr, allowed)

	if err != nil {
		return nil, err
	}

	// Soft delete şartı
	if strings.TrimSpace(whereSQL) == "" {
		whereSQL = " WHERE ISNULL(IsDeleted, 0) = 0"
	} else {
		whereSQL += " AND ISNULL(IsDeleted, 0) = 0"
	}

	// DB Connection
	db, ctx := DB.SqlOpen(c)
	defer db.Close()

	table := getTableName[T]()

	// Sorgu Oluşturma
	var query string
	queryArgs := make([]interface{}, 0, len(args)+2)
	queryArgs = append(queryArgs, args...)

	// Paging yoksa: hepsi
	if pageSize == 0 {
		query = fmt.Sprintf(`SELECT * FROM %s %s ORDER BY %s %s`, table, whereSQL, orderBy, orderDir)
	} else {
		// Paging varsa OFFSET/FETCH
		offset := (page - 1) * pageSize

		query = fmt.Sprintf(`
			SELECT *
			FROM %s
			%s
			ORDER BY %s %s
			OFFSET @offset ROWS
			FETCH NEXT @pageSize ROWS ONLY
		`, table, whereSQL, orderBy, orderDir)

		queryArgs = make([]interface{}, 0, len(args)+2)
		queryArgs = append(queryArgs, args...)
		queryArgs = append(queryArgs,
			sql.Named("offset", offset),
			sql.Named("pageSize", pageSize),
		)
	}

	// ---- Query Executer ----
	rows, err := db.QueryContext(*ctx, query, queryArgs...)
	if err != nil {
		// Eğer CreatedDate seçildi ve kolonda yoksa -> Id'ye düşlur ayni paging fonksiyonundaki gibi.
		if strings.EqualFold(orderBy, "CreatedDate") && isInvalidColumnNameErr(err) {
			// fallback (paging varsa ORDER BY Id zorunlu) Bir de ORDER BY ile denenir..
			if pageSize == 0 {
				query = fmt.Sprintf(`SELECT * FROM %s %s ORDER BY Id %s`, table, whereSQL, orderDir)
			} else {
				offset := (page - 1) * pageSize
				query = fmt.Sprintf(`
					SELECT *
					FROM %s
					%s
					ORDER BY Id %s
					OFFSET @offset ROWS
					FETCH NEXT @pageSize ROWS ONLY
				`, table, whereSQL, orderDir)

				// offset/pageSize named param zaten en sonda; yoksa zaten eklenir..
				queryArgs = make([]interface{}, 0, len(args)+2)
				queryArgs = append(queryArgs, args...)
				queryArgs = append(queryArgs,
					sql.Named("offset", offset),
					sql.Named("pageSize", pageSize),
				)

			}

			//Tekrar calisitirlir ve denenir..
			rows, err = db.QueryContext(*ctx, query, queryArgs...)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	//Tum donen rowlar alinir ve gerekli colonlar ile eslestirilir.
	for rows.Next() {
		var record T
		ptrs := make([]interface{}, len(cols))

		for i, col := range cols {
			f := getStructFieldByName(&record, col)
			if f.IsValid() && f.CanAddr() {
				ptrs[i] = f.Addr().Interface()
			} else {
				var dummy interface{}
				ptrs[i] = &dummy
			}
		}

		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}

		out = append(out, record)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Not: Boş sonuç için error döndürmüyorum. En fazla bos list donebilir..
	return out, nil
}

type Logic int

const (
	LogicAnd Logic = iota
	LogicOr
)

type Expr interface {
	isExpr()
}

// Tek koşul
type Pred struct {
	F Filter
}

// Predicate (F) => Field (F.Field) Op (F.Op) Value (F.Value)
func (Pred) isExpr() {}

// Grup (AND / OR)
type Group struct {
	Op    Logic
	Items []Expr
}

func (Group) isExpr() {}

func And(exprs ...Expr) Expr {
	return Group{Op: LogicAnd, Items: exprs}
}

func Or(exprs ...Expr) Expr {
	return Group{Op: LogicOr, Items: exprs}
}

// Düz Filter listesi = AND => Tanimli bir sey yok ise default AND
func Where(filters ...Filter) Expr {
	items := make([]Expr, 0, len(filters))
	for _, f := range filters {
		items = append(items, Pred{F: f})
	}
	return Group{Op: LogicAnd, Items: items}
}

func buildWhereFromExpr[T any](
	root Expr,
	allowed map[string]bool,
) (string, []interface{}, error) {

	if root == nil {
		return "", nil, nil
	}

	args := []interface{}{}
	paramIndex := 1

	var walk func(e Expr) (string, error)

	walk = func(e Expr) (string, error) {
		switch v := e.(type) {

		case Pred:
			field := strings.TrimSpace(v.F.Field)
			if !allowed[strings.ToLower(field)] {
				return "", fmt.Errorf("izin verilmeyen alan: %s", field)
			}

			op, err := v.F.Op.SQL()
			if err != nil {
				return "", err
			}

			p := fmt.Sprintf("p%d", paramIndex)
			paramIndex++
			args = append(args, sql.Named(p, v.F.Value))

			return fmt.Sprintf("%s %s @%s", field, op, p), nil

		case Group:
			if len(v.Items) == 0 {
				return "", nil
			}

			join := " AND "
			if v.Op == LogicOr {
				join = " OR "
			}

			parts := []string{}
			for _, it := range v.Items {
				s, err := walk(it)
				if err != nil {
					return "", err
				}
				if s != "" {
					parts = append(parts, s)
				}
			}

			if len(parts) == 0 {
				return "", nil
			}

			return "(" + strings.Join(parts, join) + ")", nil

		default:
			return "", fmt.Errorf("desteklenmeyen expr tipi")
		}
	}

	sqlExpr, err := walk(root)
	if err != nil {
		return "", nil, err
	}

	// Filter yok ise Where koymuyoruz..
	if strings.TrimSpace(sqlExpr) == "" {
		return "", args, nil
	}
	return " WHERE " + sqlExpr, args, nil
}

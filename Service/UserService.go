package Service

/*
SQL DB SCRIPT

CREATE TYPE dbo.UserTableType AS TABLE
(
    Name NVARCHAR(100),
    Age INT,
    Gender NVARCHAR(20),
    UserName NVARCHAR(100),
    Password NVARCHAR(100)
)


CREATE PROCEDURE dbo.InsertUsersBulk
    @Users dbo.UserTableType READONLY
AS
BEGIN
    INSERT INTO GOUSERS (Name, Age, Gender, UserName, Password)
    SELECT Name, Age, Gender, UserName, Password FROM @Users
END
*/
import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"httpRequestName/Core"
	"httpRequestName/DB"
	"httpRequestName/Model"
	"httpRequestName/Repository"
	shared "httpRequestName/Shared"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

func GetUserPermissions(userName string, controllerID int, actionNumber int, actionName string, c *gin.Context) (bool, error) {
	db, ctx := DB.SqlOpen(c)
	defer db.Close()
	var totalAction int = 0
	var userID int = 0
	var userNameEncrypted, errEncrypt = Core.Encrypt(userName, shared.Config.SECRETKEY)
	if errEncrypt != nil {
		return false, errEncrypt
	}
	rowUser := db.QueryRowContext(*ctx, "select Id from [dbo].[GoUsers] where  UserName= @p1 AND IsDeleted=0", userNameEncrypted)

	if rowUser.Err() != nil {
		fmt.Println(rowUser.Err().Error() + "\n")
		return false, rowUser.Err()
	}
	errUser := rowUser.Scan(&userID)
	if errUser != nil {
		fmt.Println(errUser.Error() + "\n")
		return false, errUser
	}
	row := db.QueryRowContext(*ctx, "select ActionNumberTotal from [dbo].[GoUserActions] where  IdUser= @p1 AND IdGoController =@p2 AND IsDeleted=0", userID, controllerID)

	if row.Err() != nil {
		fmt.Println(row.Err().Error() + "\n")
		return false, row.Err()
	}

	err := row.Scan(&totalAction)
	if err != nil {
		fmt.Println(err.Error() + "\n")
		return false, err
	}
	if totalAction != 0 && actionNumber == (totalAction&actionNumber) {
		return true, nil
	} else {
		//Write UnAuthorized Error
		Core.LogElasticError(userName, controllerID, actionName, "You are not Authorized!", http.StatusForbidden)
		return false, nil
	}
}

func GetUserPermissionsByRole(userName string, controllerID int, actionNumber int, actionName string, c *gin.Context) (bool, error) {
	db, ctx := DB.SqlOpen(c)
	defer db.Close()
	var totalAction int = 0
	var roleID int = 0
	var userNameEncrypted, errEncrypt = Core.Encrypt(userName, shared.Config.SECRETKEY)
	if errEncrypt != nil {
		return false, errEncrypt
	}
	rowUser := db.QueryRowContext(*ctx, "select IdRole from [dbo].[GoUsers] where  UserName= @p1 AND IsDeleted=0", userNameEncrypted)

	if rowUser.Err() != nil {
		fmt.Println(rowUser.Err().Error() + "\n")
		return false, rowUser.Err()
	}
	errUser := rowUser.Scan(&roleID)
	if errUser != nil {
		fmt.Println(errUser.Error() + "\n")
		return false, errUser
	}
	row := db.QueryRowContext(*ctx, "select ActionNumberTotal from [dbo].[GoRoleActions] where  IdGoRole= @p1 AND IdGoController =@p2 AND IsDeleted=0", roleID, controllerID)

	if row.Err() != nil {
		fmt.Println(row.Err().Error() + "\n")
		return false, row.Err()
	}

	err := row.Scan(&totalAction)
	if err != nil {
		fmt.Println(err.Error() + "\n")
		return false, err
	}
	if totalAction != 0 && actionNumber == (totalAction&actionNumber) {
		return true, nil
	} else {
		//Write UnAuthorized Error
		Core.LogElasticError(userName, controllerID, actionName, "You are not Authorized!", http.StatusForbidden)
		return false, nil
	}
}

func GetUserByUserName(name string, c *gin.Context, opts ...bool) Model.ServiceResponse[Model.Person] {
	//response := Model.ServiceResponse[Model.Person]{}
	response := Model.NewServiceResponse[Model.Person](c)
	isValidate := len(opts) > 0
	if len(opts) > 0 {
		isValidate = opts[0]
	} else {
		isValidate = true
	}
	user := Model.Person{}
	db, ctx := DB.SqlOpen(c)
	defer db.Close()
	row := db.QueryRowContext(*ctx, "select Name as name, Age as age, Gender as gender, UserName as username,Password as password, Id as id from [dbo].[GoUsers] where  Name= @p1 AND IsDeleted=0", name)

	if row.Err() != nil {
		response.Error = row.Err()
		response.Entity = user
		fmt.Println(row.Err().Error() + "\n")
		return response
	}
	err := row.Scan(&user.Name, &user.Age, &user.Gender, &user.UserName, &user.Password, &user.Id)
	if err != nil {
		//Name ile bulamaz ise UserName'e bakacagiz..updateUser icin..
		row := db.QueryRowContext(*ctx, "select Name as name, Age as age, Gender as gender, UserName as username,Password as password from [dbo].[GoUsers] where  UserName = @p1 AND IsDeleted=0", name)
		if row.Err() != nil {
			response.Error = row.Err()
			response.Entity = user
			fmt.Println(err.Error() + "\n")
			return response
		}
		err := row.Scan(&user.Name, &user.Age, &user.Gender, &user.UserName, &user.Password, &user.Id)
		if err != nil {
			fmt.Println(err.Error() + "\n")
		}
	}
	hasError := false
	if isValidate {
		for i, err := range Core.ValidateStruct(&user, true) {
			fmt.Printf("\t%d. %s\n", i+1, err.Error())
			hasError = true
		}
		if !hasError {
			response.Entity = user
			return response
		} else {
			response.Error = errors.New("model can not be decrypted")
			response.Entity = user
			return response
		}
	}
	return response
}

func FindUserByUserNameAndPassword(login Model.LoginModel, ctx *gin.Context) Model.ServiceResponse[Model.Person] {
	//response := Model.ServiceResponse[Model.Person]{}
	response := Model.NewServiceResponse[Model.Person](ctx)
	user := Model.Person{}
	hasError := false
	/*for i, err := range Core.ValidateStruct(&login) {
		fmt.Printf("\t%d. %s\n", i+1, err.Error())
		hasError = true
	}*/
	if !hasError {
		db, ctx := DB.SqlOpen(ctx)
		defer db.Close()
		row := db.QueryRowContext(*ctx, "select Name as name, Age as age, Gender as gender, UserName as username,Password as password from [dbo].[GoUsers] where  UserName= @p1 AND IsDeleted=0",
			login.Username)

		if err := row.Err(); err != nil {
			response.Error = fmt.Errorf("%w\n", err) // bu biraz alışılmadık :)
			response.Entity = user
			return response
		}

		err := row.Scan(&user.Name, &user.Age, &user.Gender, &user.UserName, &user.Password)
		if err != nil {
			response.Error = fmt.Errorf("%w\n", err)
			response.Entity = user
			return response
		}
		bytePass := []byte(login.Password)
		var isCorrectPassword = Core.ComparePasswords(user.Password, bytePass)
		fmt.Println("Is Password Correct: ", isCorrectPassword)
		if !isCorrectPassword {
			response.Error = errors.New("wrong Password")
			response.Entity = user
			return response
		}
	}
	response.Entity = user
	return response
}
func GetAllUsers(c *gin.Context) Model.ServiceResponse[Model.Person] {
	// Create a repository instance for the Person model
	repo := Repository.NewRepository[Model.Person]()
	//response := Model.ServiceResponse[Model.Person]{}
	response := Model.NewServiceResponse[Model.Person](c)
	// Call the generic GetAll method from the repository to get all users
	users, err := repo.GetAll(c)
	//users, err := repo.GetAllWithPaging(c, 2, 2, false)
	if err != nil {
		fmt.Println("Error fetching users:", err)
		response.Error = err
		return response
	}

	// Perform validation on the fetched users
	hasError := false
	for i, err := range Core.ValidateStruct(users, true) {
		fmt.Printf("\t%d. %s\n", i+1, err.Error())
		hasError = true
	}

	// If validation fails, return an error
	if hasError {
		response.Error = errors.New("models cannot be decrypted")
		return response
	}
	response.List = users
	response.Count = len(users)
	// Return the list of users if everything is okay
	return response
}

func GetUserById(c *gin.Context, id int) Model.ServiceResponse[Model.Person] {
	// Create a repository instance for the Person model
	repo := Repository.NewRepository[Model.Person]()
	//response := Model.ServiceResponse[Model.Person]{}
	response := Model.NewServiceResponse[Model.Person](c)
	// Call the generic GetAll method from the repository to get the user
	//user, err := repo.FindOne(c, Repository.Filter{Field: "Id", Op: Repository.OpEq, Value: id})
	//user, err := repo.FindOne(c, Repository.Filter{Field: "Id", Op: Repository.OpGt, Value: id})
	//encUser, err := Core.Encrypt("borsoft", shared.Config.SECRETKEY)
	//user, err := repo.FindOne(c, Repository.Filter{Field: "UserName", Op: Repository.OpEq, Value: encUser})
	user, err := repo.GetByID(c, id)
	if err != nil {
		response.Error = fmt.Errorf("failed to get user by ID: %w", err)
		return response
	}

	// Check if the returned value is nil (empty pointer)
	if user == nil {
		response.Entity = *user
		response.Error = errors.New("user not found")
		return response
	}

	// Perform validation on the fetched user
	hasError := false
	for i, err := range Core.ValidateStruct(user, true) {
		fmt.Printf("\t%d. %s\n", i+1, err.Error())
		hasError = true
	}

	// If validation fails, return an error
	if hasError {
		response.Error = errors.New("model cannot be decrypted")
		return response
	}

	// Return the user if everything is okay
	response.Entity = *user
	return response // Dereference the pointer here to return a value
}

/*func GetAllUsers() ([]Model.Person, error) {
	users := make([]Model.Person, 0)
	db, ctx := DB.SqlOpen()
	defer db.Close()

	rows, err := db.QueryContext(*ctx, "SELECT Name, Age, Gender, UserName, Password FROM [dbo].[GoUsers] WHERE IsDeleted = 0")
	if err != nil {
		fmt.Println("Query error:", err)
		return users, err
	}
	defer rows.Close()

	for rows.Next() {
		var user Model.Person
		err := rows.Scan(&user.Name, &user.Age, &user.Gender, &user.UserName, &user.Password)
		if err != nil {
			fmt.Println("Scan error:", err)
			return users, err
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		fmt.Println("Rows iteration error:", err)
		return users, err
	}
	hasError := false
	for i, err := range Core.ValidateStruct(users, true) {
		fmt.Printf("\t%d. %s\n", i+1, err.Error())
		hasError = true
	}
	if !hasError {
		return users, nil
	} else {
		return users, errors.New("models can not be decrypted")
	}
}*/

func InsertUser(user Model.Person, c *gin.Context) Model.ServiceResponse[Model.Person] {
	// Repository'yi oluştur ve Insert fonksiyonunu çağır
	repo := Repository.NewRepository[Model.Person]()
	//response := Model.ServiceResponse[Model.Person]{}
	response := Model.NewServiceResponse[Model.Person](c)

	isInserted, err := repo.Insert(user, c)
	if err != nil {
		response.Error = err
		return response
	}

	if !isInserted {
		response.Error = errors.New("insert failed")
		return response
	}

	// Serialize user to JSON for publishing
	userJSON, err := json.Marshal(user)
	if err != nil {
		log.Printf("Failed to serialize user for RabbitMQ: %v", err)
		// You can choose to return error or continue
	}

	//Add InsertedUser To RabbitMQ
	client, err := Core.NewRabbitMQClient()
	if err != nil {
		log.Fatalf("RabbitMQ connection error: %v", err)
	}
	defer client.Close()

	err = client.Publish("newUser", string(userJSON))
	if err != nil {
		log.Printf("Publish error: %v", err)
	}
	//--------RabbitMQ Publish Finish-----------

	response = GetAllUsers(c)
	if response.Error != nil {
		return response
	}
	return response
}

/*func InsertUser(user Model.Person, c *gin.Context) ([]Model.Person, error) {
hasError := false*/
//Bunu yerine Middleware'de kondu.
/*for i, err := range Core.ValidateStruct(&user) {
	fmt.Printf("\t%d. %s\n", i+1, err.Error())
	hasError = true
}*/
/*if !hasError {
		db, ctx := DB.SqlOpen(c)
		defer db.Close()

		stmt, err := db.PrepareContext(*ctx, `
		INSERT INTO GOUSERS (Name, Age, Gender, UserName, Password)
		VALUES (@p1, @p2, @p3, @p4, @p5);
	`)
		if err != nil {
			fmt.Println("Statement preparation error:", err)
			return nil, err
		}
		defer stmt.Close()

		_, err = stmt.ExecContext(*ctx,
			sql.Named("p1", user.Name),
			sql.Named("p2", user.Age),
			sql.Named("p3", user.Gender),
			sql.Named("p4", user.UserName),
			sql.Named("p5", user.Password),
		)
		if err != nil {
			fmt.Println("Statement execution error:", err)
			return nil, err
		}
	}
	users, err := GetAllUsers(c)
	if err != nil {
		return []Model.Person{}, err
	}
	return users, nil
}*/

// BulkInsertUsersTVP & Repository Sql uzerinde Type Value Parameter olusturarak StoreProcedure'e bunu parametre vererek atmak.
func BulkInsertUsersTVP(users []Model.Person, c *gin.Context) Model.ServiceResponse[Model.Person] {
	//response := Model.ServiceResponse[Model.Person]{}
	response := Model.NewServiceResponse[Model.Person](c)
	if len(users) == 0 {
		response.Error = errors.New("no users to insert")
		return response
	}

	// Kullanıcıları TVP ile eklemek için generic repository kullanılıyor
	repo := Repository.Repository[Model.Person]{}
	err := repo.BulkInsert(users, c, "dbo.UserTableType", "dbo.InsertUsersBulk", 2)
	if err != nil {
		response.Error = errors.New(fmt.Sprintf("failed to bulk insert users: %w", err))
		return response
	}

	// Tüm kullanıcıları geri dön
	response = GetAllUsers(c)
	if response.Error != nil {
		response.Error = errors.New(fmt.Sprintf("failed to retrieve users: %w", response.Error))
		return response
	}

	return response
}

// BulkInsertUsersTVP Sql uzerinde Type Value Parameter olusturarak StoreProcedure'e bunu parametre vererek atmak.
//func BulkInsertUsersTVP(users []Model.Person, c *gin.Context) ([]Model.Person, error) {
//	hasError := false
//	/*for i, err := range Core.ValidateStruct(&users) {
//		fmt.Printf("\t%d. %s\n", i+1, err.Error())
//		hasError = true
//	}*/
//	if !hasError {
//		db, ctx := DB.SqlOpen(c)
//		defer db.Close()
//
//		// Transaction başlatıyoruz
//		tx, err := db.BeginTx(*ctx, nil)
//		if err != nil {
//			return nil, fmt.Errorf("failed to begin transaction: %w", err)
//		}
//
//		//Paging Count 2ser 2ser kayit atiyor
//		const batchSize = 2
//
//		for i := 0; i < len(users); i += batchSize {
//			end := i + batchSize
//			if end > len(users) {
//				end = len(users)
//			}
//			batch := users[i:end]
//
//			tvp := mssql.TVP{
//				TypeName: "dbo.UserTableType",
//				Value:    batch,
//			}
//
//			_, err := tx.ExecContext(*ctx, "EXEC dbo.InsertUsersBulk @Users", sql.Named("Users", tvp))
//			// Transaction üzerinden sorguyu çalıştırıyoruz
//			if err != nil {
//				tx.Rollback()
//				fmt.Println("TVP Exec error:", err)
//				return nil, err
//			}
//		}
//
//		//BulkInsert'u Commit ediyoruz
//		if err := tx.Commit(); err != nil {
//			tx.Rollback()
//			return nil, fmt.Errorf("failed to commit transaction: %w", err)
//		}
//	}
//	// GetAll All Users
//	allUsers, err := GetAllUsers(c)
//	if err != nil {
//		return []Model.Person{}, err
//	}
//	/*rowsResult, err := db.QueryContext(*ctx, "SELECT Name, Age, Gender, UserName, Password FROM GOUSERS")
//	if err != nil {
//		return nil, err
//	}
//	defer rowsResult.Close()
//
//	var allUsers []Model.Person
//	for rowsResult.Next() {
//		var u Model.Person
//		if err := rowsResult.Scan(&u.Name, &u.Age, &u.Gender, &u.UserName, &u.Password); err != nil {
//			return nil, err
//		}
//		allUsers = append(allUsers, u)
//	}
//	*/
//
//	return allUsers, nil
//}

func BulkInsert(users []Model.Person, c *gin.Context, batchSize int) Model.ServiceResponse[Model.Person] {
	repo := Repository.NewRepository[Model.Person]()
	response := Model.NewServiceResponse[Model.Person](c)
	//var response = Model.ServiceResponse[Model.Person]{}
	_, err := repo.BulkInsertBatched(users, c, batchSize)
	if err != nil {
		response.Error = err
		return response
	}
	response = GetAllUsers(c)
	return response
}

// 100er 100er Insert Query olusturup atmak
func BulkInsertUsersBatched(users []Model.Person, c *gin.Context) Model.ServiceResponse[Model.Person] {
	//var response = Model.ServiceResponse[Model.Person]{}
	response := Model.NewServiceResponse[Model.Person](c)
	hasError := false
	for i, err := range Core.ValidateStruct(&users) {
		fmt.Printf("\t%d. %s\n", i+1, err.Error())
		hasError = true
	}
	if !hasError {
		db, ctx := DB.SqlOpen()
		defer db.Close()

		// Transaction başlatıyoruz
		tx, err := db.BeginTx(*ctx, nil)
		if err != nil {
			response.Error = fmt.Errorf("failed to begin transaction: %w", err)
			return response
		}

		// Paging Count: 2'şer 2'şer kayıt atıyor
		const batchSize = 2
		paramCounter := 1

		for i := 0; i < len(users); i += batchSize {
			end := i + batchSize
			if end > len(users) {
				end = len(users)
			}
			batch := users[i:end]
			/*if end == 3 {
				errors.New("Custom Error Execption")
				tx.Rollback()
			}*/
			query := "INSERT INTO GOUSERS (Name, Age, Gender, UserName, Password) VALUES "
			var placeholders []string
			var args []interface{}

			for _, user := range batch {
				placeholders = append(placeholders, fmt.Sprintf("(@p%d, @p%d, @p%d, @p%d, @p%d)",
					paramCounter, paramCounter+1, paramCounter+2, paramCounter+3, paramCounter+4))

				args = append(args,
					sql.Named(fmt.Sprintf("p%d", paramCounter), user.Name),
					sql.Named(fmt.Sprintf("p%d", paramCounter+1), user.Age),
					sql.Named(fmt.Sprintf("p%d", paramCounter+2), user.Gender),
					sql.Named(fmt.Sprintf("p%d", paramCounter+3), user.UserName),
					sql.Named(fmt.Sprintf("p%d", paramCounter+4), user.Password),
				)

				paramCounter += 5
			}

			query += strings.Join(placeholders, ", ")

			// Transaction üzerinden sorguyu çalıştırıyoruz
			_, err := tx.ExecContext(*ctx, query, args...)
			if err != nil {
				tx.Rollback()
				response.Error = fmt.Errorf("batch insert failed: %w", err)
				return response
			}
		}

		//BulkInsert'u Commit ediyoruz
		if err := tx.Commit(); err != nil {
			tx.Rollback()
			response.Error = fmt.Errorf("failed to commit transaction: %w", err)

			return response
		}
	}
	// Tum Kullanicilari Cekiyoruz. Transaction disinda.
	response = GetAllUsers(c)
	if response.Error != nil {
		return response
	}
	/*rowsResult, err := db.QueryContext(*ctx, "SELECT Name, Age, Gender, UserName, Password FROM GOUSERS")
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rowsResult.Close()

	var allUsers []Model.Person
	for rowsResult.Next() {
		var u Model.Person
		if err := rowsResult.Scan(&u.Name, &u.Age, &u.Gender, &u.UserName, &u.Password); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		allUsers = append(allUsers, u)
	}*/

	return response
}

var updateLock sync.Mutex

func UpdateUser(newUser Model.Person, c *gin.Context) Model.ServiceResponse[Model.Person] {
	//var response = Model.ServiceResponse[Model.Person]{}
	response := Model.NewServiceResponse[Model.Person](c)
	// Kilitleme işlemi burada yapılacak
	updateLock.Lock()
	defer updateLock.Unlock()

	// Kullanıcı adını şifrele
	encryptedUserName, errEncrypt := Core.Encrypt(newUser.UserName, shared.Config.SECRETKEY)
	if errEncrypt != nil {
		response.Error = errEncrypt
		return response
	}

	// Eski kullanıcıyı al
	oldUserResponse := GetUserByUserName(encryptedUserName, c, true)
	if oldUserResponse.Error != nil {
		response.Error = oldUserResponse.Error
		return response
	}

	// Repository'yi oluştur ve Update fonksiyonunu çağır
	repo := Repository.NewRepository[Model.Person]()
	updated, err := repo.Update(oldUserResponse.Entity, newUser, encryptedUserName, c)
	if err != nil {
		response.Error = err
		return response
	}

	if !updated {
		response.Error = fmt.Errorf("update failed")
		return response
	}
	response.Entity = newUser
	return response
}

func UpdateAllUsersAges(updateColumns map[string]interface{}, filters map[string]interface{}, c *gin.Context) Model.ServiceResponse[Model.Person] {
	repo := Repository.NewRepository[Model.Person]()
	//var response = Model.ServiceResponse[Model.Person]{}
	response := Model.NewServiceResponse[Model.Person](c)
	success, err := repo.BulkUpdateWithFilter(
		updateColumns,
		filters,
		c,
	)
	if err != nil {
		response.Error = err
		return response
	}

	if !success {
		response.Error = fmt.Errorf("Users Update failed")
		return response
	}
	response = GetAllUsers(c)
	/*if response.IsError != true {
		persons = []Model.Person{}
	}*/
	return response
}

/*func UpdateUser(newUser Model.Person, c *gin.Context) (bool, error) {
	updateLock.Lock()
	defer updateLock.Unlock()

	db, ctx := DB.SqlOpen(c)
	defer db.Close()

	var encryptedUserName, errEncrypt = Core.Encrypt(newUser.UserName, shared.Config.SECRETKEY)
	if errEncrypt != nil {
		return false, errEncrypt
	}
	// 1. First, get the old user from database
	oldUser, err := GetUserByUserName(encryptedUserName, c, true)
	if err != nil {
		return false, err
	}

	// 2. Detect changed fields
	changes := Core.GetChangedFields(oldUser, newUser)
	if len(changes) == 0 {
		fmt.Println("No changes detected.")
		return true, nil
	}

	// 3. Build dynamic SQL
	query := "UPDATE GOUSERS SET "
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
	args = append(args, sql.Named(fmt.Sprintf("p%d", i), newUser.UserName))

	// 4. Prepare and execute
	stmt, err := db.PrepareContext(*ctx, query)
	if err != nil {
		fmt.Println("Statement preparation error:", err)
		return false, err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(*ctx, args...)
	if err != nil {
		fmt.Println("Statement execution error:", err)
		return false, err
	}

	return true, nil
}*/

var batchUpdateLock sync.Mutex

// BatchUpdatePersons birden fazla Person'ı UserName'e göre günceller.
// - Önce tüm işlem bir mutex ile kilitlenir.
// - Her yeni kayıttaki UserName, validate:"encrypt" tag'ı kontrol edilerek şifrelenir.
// - Şifrelenmiş UserName'e göre eski veri çekilir.
// - Core.GetChangedFields ile değişiklikler bulunur ve generic BatchUpdate çağrılır.
func BatchUpdatePersons(c *gin.Context, newEntities []Model.Person, batchSize int) Model.ServiceResponse[Model.Person] {
	//var response = Model.ServiceResponse[Model.Person]{}
	response := Model.NewServiceResponse[Model.Person](c)
	batchUpdateLock.Lock()
	defer batchUpdateLock.Unlock()

	// 1) Açılan DB bağlantısı sadece sorgu için
	db, ctx := DB.SqlOpen(c)
	defer db.Close()

	// 2) Hazırlanacak pair listesi
	var pairs []struct{ Old, New Model.Person }

	for _, newEntity := range newEntities {
		// 2a) Şifreleme (sadece UserName için)
		encryptedUserName, err := Core.Encrypt(newEntity.UserName, shared.Config.SECRETKEY)
		if err != nil {
			response.Error = fmt.Errorf("username encrypt error: %w", err)
			return response
		}
		newEntity.UserName = encryptedUserName

		// 2b) Eski kaydı çek
		var oldEntity Model.Person
		query := "SELECT name, age, gender, username, password FROM GoUsers WHERE UserName = @p1 AND ISNULL(IsDeleted,0)=0"
		if err := db.QueryRowContext(*ctx, query, sql.Named("p1", newEntity.UserName)).
			Scan(&oldEntity.Name, &oldEntity.Age, &oldEntity.Gender, &oldEntity.UserName, &oldEntity.Password); err != nil {

			if errors.Is(err, sql.ErrNoRows) {
				// Yeni kayıt DB'de yoksa güncelleme listesine dahil etme
				continue
			}
			response.Error = fmt.Errorf("DB query failed: %w", err)
			return response
		}

		pairs = append(pairs, struct{ Old, New Model.Person }{Old: oldEntity, New: newEntity})
	}

	// 3) Bulk update işlemi (repository içinde kendi bağlantısını açar)
	repo := Repository.NewRepository[Model.Person]()
	if err := repo.BatchUpdate(pairs, c, "UserName", batchSize); err != nil {
		response.Error = fmt.Errorf("batch update failed: %w", err)
		return response
	}
	response.Message = "Batch update successful"
	return response
}

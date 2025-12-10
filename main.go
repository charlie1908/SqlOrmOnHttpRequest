package main

/*
REDIS SET PASSWORD WITH CLI
CONFIG SET requirepass fl@rp1$19C23

SWAGGER
go get -u github.com/swaggo/gin-swagger
go get -u github.com/swaggo/files
go install github.com/swaggo/swag/cmd/swag@latest
swag init
----------
go get -u github.com/gin-gonic/gin
*/

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"httpRequestName/Core"
	"httpRequestName/Interceptors"
	"httpRequestName/Model"
	"httpRequestName/RabbitMQ"
	"httpRequestName/Service"
	shared "httpRequestName/Shared"
	_ "httpRequestName/docs" // Swagger dokümantasyonunu içe aktarıyoruz
	"io"
	"log"
	"net/http"
	"strconv"
)

// @title           Person API
// @version         1.0
// @description     Bu API, kişileri listelemek, eklemek ve isim bazında sorgulamak için kullanılır.
// @host           localhost:1923
// @BasePath       /

// persons listesi
// var persons = make([]Model.Person, 0)
var token = ""

func main() {
	/*var Url, _ = Core.Decrypt(shared.Config.REDISURL, shared.Config.SECRETKEY)
	var password, _ = Core.Decrypt(shared.Config.REDISPASSWORD, shared.Config.SECRETKEY)
	var passwordElastic, _ = Core.Decrypt(shared.Config.ELASTICPASSWORD, shared.Config.SECRETKEY)
	var passwordRabbit, _ = Core.Decrypt(shared.Config.RABBITMQPASSWORD, shared.Config.SECRETKEY)
	fmt.Println(Url)
	//fmt.Println(urlLocal)
	fmt.Println(password)
	fmt.Println(passwordElastic)
	fmt.Println(passwordRabbit)*/

	/*persons = []Model.Person{
		{Name: "Duru", Age: 13, Gender: Model.GenderType.Female},
		{Name: "Arya", Age: 7, Gender: Model.GenderType.Female},
		{Name: "Secil", Age: 42, Gender: Model.GenderType.Female},
		{Name: "Bora", Age: 47, Gender: Model.GenderType.Male},
	}*/

	// 1. RabbitMQ listener'ı ayrı bir goroutine'de başlat
	go func() {
		err := RabbitMQ.ListenUserRabbitMQ()
		if err != nil {
			log.Fatalf("RabbitMQ dinleyicisinde hata: %v", err)
		}
	}()

	//Multi language diller yuklenir.
	Core.InitTranslator()

	router := gin.Default()

	//Set GlobalFilter With Redis Store
	//1000 Request Per Minute
	globalLimiter := Core.GetGlobalLimiter()
	router.Use(Interceptors.GlobalRateLimitMiddleware(globalLimiter))

	// Swagger endpointini ekleyelim
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	router.GET("/persons", getPersons)
	//router.GET("/person/:name", Interceptors.AuthMiddleware(Core.ModulesEnum.Users, Core.UserEnumToString[Core.UserActionEnum.GetUserById]), getPersonByName)
	router.GET("/person/name/:name", Interceptors.AuthMiddleware(Core.ModulesEnum.Users, Core.GetEnumName(Core.UserActionEnum, Core.UserActionEnum.GetUserById)),
		Interceptors.PermissionCheck(Core.ModulesEnum.Users, Core.UserActionEnum.GetUserById, Core.GetEnumName(Core.UserActionEnum, Core.UserActionEnum.GetUserById)), getPersonByName)
	router.GET("/person/id/:id", Interceptors.AuthMiddleware(Core.ModulesEnum.Users, Core.GetEnumName(Core.UserActionEnum, Core.UserActionEnum.GetUserById)), getPersonById)
	//router.POST("/person", Interceptors.AuthMiddleware(Core.ModulesEnum.Users, Core.UserEnumToString[Core.UserActionEnum.InsertUser]),
	router.POST("/person", Interceptors.AuthMiddleware(Core.ModulesEnum.Users, Core.GetEnumName(Core.UserActionEnum, Core.UserActionEnum.InsertUser)),
		Interceptors.LoggingMiddleware(), Interceptors.InsertAuditLogMiddleware("Insert", "GoUsers"),
		Interceptors.TransformMiddleware[Model.Person](), insertPerson)
	router.POST("/updatePerson", Interceptors.AuthMiddleware(Core.ModulesEnum.Users, Core.GetEnumName(Core.UserActionEnum, Core.UserActionEnum.UpdateUser)),
		Interceptors.PermissionCheck(Core.ModulesEnum.Users, Core.UserActionEnum.UpdateUser, Core.GetEnumName(Core.UserActionEnum, Core.UserActionEnum.UpdateUser)),
		updatePerson)
	router.PATCH("/updatePersonsField/:field/:op/:value",
		Interceptors.AuthMiddleware(Core.ModulesEnum.Users, Core.GetEnumName(Core.UserActionEnum, Core.UserActionEnum.UpdateUser)),
		updatePersonsField)
	router.POST(
		"/updatePersons",
		Interceptors.AuthMiddleware(
			Core.ModulesEnum.Users,
			Core.GetEnumName(Core.UserActionEnum, Core.UserActionEnum.UpdateUser),
		),
		Interceptors.PermissionCheck(
			Core.ModulesEnum.Users,
			Core.UserActionEnum.UpdateUser,
			Core.GetEnumName(Core.UserActionEnum, Core.UserActionEnum.UpdateUser),
		),
		updatePersons,
	)

	//router.POST("/persons", Interceptors.AuthMiddleware(Core.ModulesEnum.Users, Core.UserEnumToString[Core.UserActionEnum.BulkInsertUser]),
	router.POST("/persons", Interceptors.AuthMiddleware(Core.ModulesEnum.Users,
		Core.GetEnumName(Core.UserActionEnum, Core.UserActionEnum.BulkInsertUser)),
		Interceptors.PermissionCheck(Core.ModulesEnum.Users, Core.UserActionEnum.BulkInsertUser,
			Core.GetEnumName(Core.UserActionEnum, Core.UserActionEnum.BulkInsertUser)),
		Interceptors.LoggingMiddleware(), Interceptors.TransformMiddleware[Model.Person](),
		Interceptors.InsertAuditLogMiddleware("Insert", "GoUsers"), bulkInsertPerson)
	router.POST("/login", Interceptors.TransformMiddleware[Model.LoginModel](), Interceptors.InsertLoginLogMiddleware(), login)

	//var err = router.Run(":1923")
	var err = router.Run("0.0.0.0:1923")
	if err != nil {
		panic(err)
	}
}

// getPersons tüm kişileri döndürür
// @Summary      Tüm kişileri getir
// @Description  Veritabanındaki tüm kişileri getirir
// @Tags         persons
// @Produce      json
// @Success      200  {array}  Model.Person
// @Router       /persons [get]
func getPersons(c *gin.Context) {
	response := Service.GetAllUsers(c)
	if response.Error != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "persons not found"})
	}
	c.IndentedJSON(http.StatusOK, response)
}

// getPersonByName isme göre kişiyi döndürür
// @Summary      Belirli bir isme sahip kişiyi getir
// @Description  İstenilen isme sahip kişiyi getirir
// @Tags         persons
// @Param        name   path      string  true  "Kişi İsmi"
// @Produce      json
// @Success      200  {object}  Model.Person
// @Failure      404  {object}  map[string]string
// @Router       /person/name/{name} [get]
func getPersonByName(c *gin.Context) {
	name := c.Param("name")
	var response = Service.GetUserByUserName(name, c)
	/*var result = Model.Filters(persons, func(person Model.Person) bool {
		return strings.EqualFold(person.Name, name)
	})
	if result != nil {
		c.IndentedJSON(http.StatusOK, result)
		return
	}*/
	if response.Error == nil {
		//c.IndentedJSON(http.StatusOK, response.Entity)
		c.IndentedJSON(http.StatusOK, response)
		return
	}
	c.IndentedJSON(http.StatusNotFound, gin.H{"message": "person not found"})
}

// getPersonById isme göre kişiyi döndürür
// @Summary      Belirli bir ID'ye sahip kişiyi getir
// @Description  İstenilen ID'ye sahip kişiyi getirir
// @Tags         persons
// @Param        id   path      int  true  "Kişi Id'si"
// @Produce      json
// @Success      200  {object}  Model.Person
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /person/id/{Id} [get]
func getPersonById(c *gin.Context) {
	idParam := c.Param("id")

	// String'i int'e çevir
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}

	// Servis çağrısı
	response := Service.GetUserById(c, id)
	if response.Error == nil {
		c.IndentedJSON(http.StatusOK, response)
		return
	}

	c.IndentedJSON(http.StatusNotFound, gin.H{"message": "person not found"})
}

// insertPerson yeni bir kişi ekler
// @Summary      Yeni bir kişi ekle
// @Description  JSON formatında yeni bir kişi ekler
// @Tags         persons
// @Accept       json
// @Produce      json
// @Param        person  body      Model.Person  true  "Yeni kişi"
// @Success      201  {object}  Model.Person
// @Failure      400  {object}  map[string]string
// @Router       /person [post]
// func insertPerson(c *gin.Context) {
func insertPerson(c *gin.Context) {
	raw, exists := c.Get("payload")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing person in context"})
		return
	}
	/*var person = Model.Person{}
	if err := c.ShouldBindJSON(&person); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}*/
	// Buraya gelen person'in zaten Sensitive datalari bir onceki middleware functioninda gizlenmis olarak hazir geliyor..
	person := raw.(Model.Person)
	//persons = append(persons, person)
	response := Service.InsertUser(person, c)
	if response.Error != nil {
		//Write Error Log to Elastic
		_username := c.GetHeader("UserName")
		//Core.LogElasticError(_username, Core.ModulesEnum.Users, Core.UserEnumToString[Core.UserActionEnum.InsertUser], err.Error(), 500)
		Core.LogElasticError(_username, Core.ModulesEnum.Users, Core.GetEnumName(Core.UserActionEnum, Core.UserActionEnum.InsertUser), response.Error.Error(), 500)
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": response.Error.Error()})
	}
	c.IndentedJSON(http.StatusCreated, response)
}

/*
TEST SWAGGER DATA
[
  {
    "age": 47,
    "gender": "Male",
    "name": "Bora",
    "username": "borsoft",
    "password": "123456"
  },
  {
    "age": 13,
    "gender": "Female",
    "name": "Duru",
    "username": "dursoft",
    "password": "123456"
  },
  {
    "age": 42,
    "gender": "Female",
    "name": "Secil",
    "username": "secsoft",
    "password": "123456"
  },
  {
    "age": 7,
    "gender": "Female",
    "name": "Arya",
    "username": "arsoft",
    "password": "123456"
  }
]
*/

// bulkInsertPerson birden fazla kişi ekler
// @Summary      Birden fazla kişi ekle
// @Description  JSON formatında birden fazla yeni kişi ekler
// @Tags         persons
// @Accept       json
// @Produce      json
// @Param        persons  body      []Model.Person  true  "Birden fazla yeni kişi"
// @Success      201  {array}  Model.Person  "Başarıyla eklenen kişiler"
// @Failure      400  {object}  map[string]string  "Geçersiz giriş verisi"
// @Router       /persons [post]
func bulkInsertPerson(c *gin.Context) {
	raw, exists := c.Get("payload")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing person in context"})
		return
	}
	persons := raw.([]Model.Person)
	/*var persons []Model.Person
	if err := c.ShouldBindJSON(&persons); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}*/
	//persons = append(persons, person)
	//persons, err := Service.BulkInsertUsersTVP(persons, c) //Use Repository BulkInsertUsersTVP
	response := Service.BulkInsert(persons, c, 100) //Use Repository
	//persons, err := Service.BulkInsertUsersBatched(persons,c) //Use Directly BulkInsertUsersBatched() function
	if response.Error != nil {
		_username := c.GetHeader("UserName")
		//Core.LogElasticError(_username, Core.ModulesEnum.Users, Core.UserEnumToString[Core.UserActionEnum.BulkInsertUser], err.Error(), 500)
		Core.LogElasticError(_username, Core.ModulesEnum.Users, Core.GetEnumName(Core.UserActionEnum, Core.UserActionEnum.BulkInsertUser), response.Error.Error(), 500)
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": response.Error.Error()})
	}
	c.IndentedJSON(http.StatusCreated, response)
}

// login kullanıcı girişi yapar
// @Summary      Kullanıcı Girişi
// @Description  Kullanıcı adı ve şifre ile giriş yapar
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        credentials  body      Model.LoginModel  true  "Kullanıcı Girişi"
// @Success      200  {object}  map[string]string  "Başarılı giriş"
// @Failure      400  {object}  map[string]string  "Geçersiz giriş verisi"
// @Failure      401  {object}  map[string]string  "Yetkisiz erişim"
// @Router       /login [post]
func login(c *gin.Context) {
	raw, exists := c.Get("payload")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing person in context"})
		return
	}
	/*var login Model.LoginModel
	if err := c.ShouldBindJSON(&login); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}*/
	login := raw.(Model.LoginModel)
	c.Set("UserName", login.Username)
	findUserResponse := Service.FindUserByUserNameAndPassword(login, c)
	if findUserResponse.Error != nil {
		//Insert Elastic To Error Login. Attempt To Login...
		// Audit log verisi hazırla
		var _username, _ = Core.Decrypt(login.Username, shared.Config.SECRETKEY)
		//Core.LogElasticError(_username, Core.ModulesEnum.Login, Core.LoginEnumToString[Core.LoginActionEnum.Login], "401 Unauthorized", 401)
		Core.LogElasticError(_username, Core.ModulesEnum.Login, Core.GetEnumName(Core.LoginActionEnum, Core.LoginActionEnum.Login), "401 Unauthorized", 401)

		/*var _username, _ = Core.Decrypt(login.Username, shared.Config.SECRETKEY)
		errorData := Model.ErrorLog{
			UserName:   _username,
			ModuleName: Core.ModuleEnumToString[Core.ModulesEnum.Login],
			ActionName: Core.LoginEnumToString[Core.LoginActionEnum.Login],
			Message:    "401 Unauthorized",
			DateTime:   time.Now(),
			ErrorCode:  401,
		}

		// Logu ElasticSearch'e gönder
		if err := Core.InsertLog(errorData, shared.Config.ELASTICERRORINDEX); err != nil {
			log.Println("Login 401 Unauthorized Log Error:", err)
			// Gecersiz Login denemesi Elastic'e kaydedilir.
		}*/
		//-------------
		c.IndentedJSON(http.StatusUnauthorized, gin.H{
			"error":   "wrong username or password",
			"details": findUserResponse.Error.Error(),
		})

		return
	}

	token = Model.GenerateToken(36)
	refreshToken := Model.GenerateToken(36)
	redisClient := Core.GetRedisClient()
	userName, _ := Core.Decrypt(login.Username, shared.Config.SECRETKEY)
	var err2 = redisClient.SetKey(Core.GenerateRedisKey(userName, false), token, shared.Config.TOKENEXPIRETIME) // SET TOKEN TO THE REDIS WITH EXPIRETIME
	if err2 != nil {
		c.IndentedJSON(http.StatusUnauthorized, gin.H{"message": "wrong username or password"})
		return
	}
	//RefreshToken
	var err3 = redisClient.SetKey(Core.GenerateRedisKey(userName, true), refreshToken, shared.Config.REFRESHTOKENEXPIRETIME) // SET REFRESHTOKEN TO THE REDIS WITH EXPIRETIME
	if err3 != nil {
		c.IndentedJSON(http.StatusUnauthorized, gin.H{"message": "wrong username or password"})
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{"message": "you are logged in " + findUserResponse.Entity.Name, "token": token, "refreshToken": refreshToken, "userID": login.Username})
	return
}

// updatePerson yeni bir kişi ekler
// @Summary      Yeni bir kişi ekle
// @Description  JSON formatında yeni bir kişi ekler
// @Tags         persons
// @Accept       json
// @Produce      json
// @Param        person  body      Model.Person  true  "Yeni kişi"
// @Success      201  {object}  Model.Person
// @Failure      400  {object}  map[string]string
// @Router       /person [post]
// func insertPerson(c *gin.Context) {
func updatePerson(c *gin.Context) {
	var person Model.Person
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		c.Abort()
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Tekrar kullanılabilir hale getir

	if err := json.Unmarshal(bodyBytes, &person); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON: " + err.Error(),
		})
	}

	//persons = append(persons, person)
	response := Service.UpdateUser(person, c)
	if response.Error != nil {
		//Write Error Log to Elastic
		_username := c.GetHeader("UserName")
		//Core.LogElasticError(_username, Core.ModulesEnum.Users, Core.UserEnumToString[Core.UserActionEnum.InsertUser], err.Error(), 500)
		Core.LogElasticError(_username, Core.ModulesEnum.Users, Core.GetEnumName(Core.UserActionEnum, Core.UserActionEnum.UpdateUser), response.Error.Error(), 500)
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": response.Error.Error()})
	}
	c.IndentedJSON(http.StatusCreated, response)
}

// updatePersonsField bir alanın değerini toplama, çıkarma, çarpma, bölme veya sabit atama ile günceller
// @Summary      Alan güncelleme
// @Description  Belirli bir alanı verilen operasyona göre günceller. (örneğin: Age alanını +1 yap)
// @Tags         persons
// @Accept       json
// @Produce      json
// @Param        field  path     string  true  "Güncellenecek alan (örneğin: Age)"
// @Param        op     path     string  true  "İşlem türü (add, sub, mul, div, set)"
// @Param        value  path     string  true  "Uygulanacak değer (örneğin: 5)"
// @Success      200    {array}   Model.Person
// @Failure      400    {object}  map[string]string
// @Router       /persons/update-field [patch]
// updatePersonsField updates a field's value based on the specified operation (e.g., add, sub, mul, div, set)
func updatePersonsField(c *gin.Context) {
	field := c.Param("field")
	op := c.Param("op")
	value := c.Param("value")

	if field == "" || op == "" || value == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "missing parameters"})
		return
	}

	// ✅ Güvenli alanlar listesi (yalnızca bu alanlara izin verilecek)
	allowedFields := map[string]bool{
		"Age":       true,
		"Gender":    true,
		"IsDeleted": true,
		// diğer izin verilen kolonları buraya ekle
	}

	if !allowedFields[field] {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "invalid field"})
		return
	}

	// Sayısal değer kontrolü (sadece add/sub/mul/div için)
	if op != "set" {
		if _, err := strconv.Atoi(value); err != nil {
			c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "value must be numeric"})
			return
		}
	}

	// SQL ifadesini oluştur
	var expr string
	switch op {
	case "add":
		expr = fmt.Sprintf("%s + %s", field, value)
	case "sub":
		expr = fmt.Sprintf("%s - %s", field, value)
	case "mul":
		expr = fmt.Sprintf("%s * %s", field, value)
	case "div":
		expr = fmt.Sprintf("%s / %s", field, value)
	case "set":
		expr = value // doğrudan sabit değer
	default:
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "invalid operation"})
		return
	}

	var updateData map[string]interface{}
	if op == "set" {
		updateData = map[string]interface{}{field: value}
	} else {
		updateData = map[string]interface{}{field: Model.RawValue{Expr: expr}}
	}

	response := Service.UpdateAllUsersAges(
		updateData,
		map[string]interface{}{
			"IsDeleted": 0,
			"Gender":    "Male",
		},
		c,
	)

	if response.Error != nil {
		username := c.GetHeader("UserName")
		Core.LogElasticError(username, Core.ModulesEnum.Users, Core.GetEnumName(Core.UserActionEnum, Core.UserActionEnum.UpdateUser), fmt.Sprintf("updateFieldValue failed. err: %v", response.Error.Error()), 500)
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "update failed"})
		return
	}

	c.IndentedJSON(http.StatusOK, response)
}

// updatePersons birden fazla Person'ı günceller
// @Summary      Birden fazla kişiyi güncelle
// @Description  JSON formatında gönderilen kullanıcıları topluca günceller (UserName alanı encrypt edilerek)
// @Tags         persons
// @Accept       json
// @Produce      json
// @Param        persons  body      []Model.Person  true  "Güncellenecek kişiler listesi"
// @Success      200      {object}  map[string]string
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /updatePersons [post]
func updatePersons(c *gin.Context) {
	response := Model.NewServiceResponse[Model.Person](c)
	var newEntities []Model.Person
	if err := c.ShouldBindJSON(&newEntities); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON: " + err.Error()})
		return
	}

	// Servis çağrısı; artık identifierField parametresi alınmıyor
	if response = Service.BatchUpdatePersons(c, newEntities, 100); response.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed: " + response.Error.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, response)
}

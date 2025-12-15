package docs

// Define the custom Swagger doc template with your custom security definitions and other fields.
const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "securityDefinitions": {
        "BearerAuth": {
            "type": "apiKey",
            "name": "Authorization",
            "in": "header"
        }
    },
    "paths": {
        "/login": {
            "post": {
                "description": "Kullanıcı adı ve şifre ile giriş yapar",
                "consumes": ["application/json"],
                "produces": ["application/json"],
                "tags": ["auth"],
                "summary": "Kullanıcı Girişi",
                "parameters": [
                    {
                        "description": "Kullanıcı Girişi",
                        "name": "credentials",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/Model.LoginModel"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Başarılı giriş",
                        "schema": { "type": "object", "additionalProperties": { "type": "string" } }
                    },
                    "400": {
                        "description": "Geçersiz giriş verisi",
                        "schema": { "type": "object", "additionalProperties": { "type": "string" } }
                    },
                    "401": {
                        "description": "Yetkisiz erişim",
                        "schema": { "type": "object", "additionalProperties": { "type": "string" } }
                    }
                }
            }
        },
        "/person": {
            "post": {
                "security": [{ "BearerAuth": [] }],
                "description": "JSON formatında yeni bir kişi ekler",
                "consumes": ["application/json"],
                "produces": ["application/json"],
                "tags": ["persons"],
                "summary": "Yeni bir kişi ekle",
                "parameters": [
                    {
                        "description": "Yeni kişi",
                        "name": "person",
                        "in": "body",
                        "required": true,
                        "schema": { "$ref": "#/definitions/Model.Person" }
                    },
                    {
                        "description": "User Name",
                        "name": "userName",
                        "in": "header",
                        "required": true,
                        "type": "string"
                    },
                    {
                        "description": "Refresh Token",
                        "name": "refreshToken",
                        "in": "header",
                        "required": false,
                        "type": "string"
                    }
                ],
                "responses": {
                    "201": { "description": "Created", "schema": { "$ref": "#/definitions/ServiceResponsePerson" } },
                    "400": { "description": "Bad Request", "schema": { "type": "object", "additionalProperties": { "type": "string" } } }
                }
            }
        },
        "/updatePerson": {
            "post": {
                "security": [{ "BearerAuth": [] }],
                "description": "Varolan bir kişinin bilgilerini günceller",
                "consumes": ["application/json"],
                "produces": ["application/json"],
                "tags": ["persons"],
                "summary": "Bir kişiyi güncelle (POST)",
                "parameters": [
                    {
                        "description": "Güncellenecek kişi bilgileri",
                        "name": "person",
                        "in": "body",
                        "required": true,
                        "schema": { "$ref": "#/definitions/Model.Person" }
                    },
                    {
                        "description": "User Name",
                        "name": "userName",
                        "in": "header",
                        "required": true,
                        "type": "string"
                    },
                    {
                        "description": "Refresh Token",
                        "name": "refreshToken",
                        "in": "header",
                        "required": false,
                        "type": "string"
                    }
                ],
                "responses": {
                    "200": { "description": "Güncellenen kişi bilgileri", "schema": { "$ref": "#/definitions/ServiceResponsePerson" } },
                    "400": { "description": "Geçersiz istek", "schema": { "type": "object", "additionalProperties": { "type": "string" } } },
                    "404": { "description": "Kişi bulunamadı", "schema": { "type": "object", "additionalProperties": { "type": "string" } } }
                }
            }
        },
"/updatePersonsField/{field}/{op}/{value}": {
  "patch": {
    "security": [{ "BearerAuth": [] }],
    "description": "Belirli bir alanı verilen operasyona göre günceller. (örneğin: Age alanını +1 yap)",
    "consumes": ["application/json"],
    "produces": ["application/json"],
    "tags": ["persons"],
    "summary": "Alan güncelleme",
    "parameters": [
      {
        "name": "field",
        "in": "path",
        "description": "Güncellenecek alan (örneğin: Age)",
        "required": true,
        "type": "string"
      },
      {
        "name": "op",
        "in": "path",
        "description": "İşlem türü (add, sub, mul, div, set)",
        "required": true,
        "type": "string"
      },
      {
        "name": "value",
        "in": "path",
        "description": "Uygulanacak değer (örneğin: 5)",
        "required": true,
        "type": "string"
      },
      {
        "name": "userName",
        "in": "header",
        "description": "User Name",
        "required": true,
        "type": "string"
      },
      {
        "name": "refreshToken",
        "in": "header",
        "description": "Refresh Token",
        "required": false,
        "type": "string"
      }
    ],
    "responses": {
      "200": {
        "description": "Başarılı işlem",
        "schema": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/ServiceResponsePerson"
          }
        }
      },
      "400": {
        "description": "Geçersiz istek",
        "schema": {
          "type": "object",
          "additionalProperties": {
            "type": "string"
          }
        }
      }
    }
  }
},
"/updatePersons": {
  "post": {
    "security": [{ "BearerAuth": [] }],
    "description": "Birden fazla kişinin bilgilerini toplu olarak günceller. Sadece değişiklik olan alanlar güncellenir.",
    "consumes": ["application/json"],
    "produces": ["application/json"],
    "tags": ["persons"],
    "summary": "Toplu kişi güncelleme",
    "parameters": [
      {
        "description": "Güncellenecek kişiler listesi",
        "name": "persons",
        "in": "body",
        "required": true,
        "schema": {
          "type": "array",
          "items": { "$ref": "#/definitions/Model.Person" }
        }
      },
      {
        "description": "User Name",
        "name": "userName",
        "in": "header",
        "required": true,
        "type": "string"
      },
      {
        "description": "Refresh Token",
        "name": "refreshToken",
        "in": "header",
        "required": false,
        "type": "string"
      }
    ],
    "responses": {
      "200": {
        "description": "Başarılı güncelleme",
        "schema": {
          "type": "object",
          "properties": {
            "message": { "type": "string" }
          }
        }
      },
      "400": {
        "description": "Geçersiz istek",
        "schema": {
          "type": "object",
          "additionalProperties": { "type": "string" }
        }
      },
      "500": {
        "description": "Sunucu hatası",
        "schema": {
          "type": "object",
          "additionalProperties": { "type": "string" }
        }
      }
    }
  }
},
        "/person/id/{id}": {
            "get": {
                "security": [{ "BearerAuth": [] }],
                "description": "ID ile kişiyi getirir",
                "produces": ["application/json"],
                "tags": ["persons"],
                "summary": "Belirli bir ID'ye sahip kişiyi getir",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "Kişi ID'si",
                        "name": "id",
                        "in": "path",
                        "required": true
                    },
                    {
                        "description": "User Name",
                        "name": "userName",
                        "in": "header",
                        "required": true,
                        "type": "string"
                    },
                    {
                        "description": "Refresh Token",
                        "name": "refreshToken",
                        "in": "header",
                        "required": false,
                        "type": "string"
                    }
                ],
                "responses": {
                    "200": { "description": "OK", "schema": { "$ref": "#/definitions/ServiceResponsePerson" } },
                    "404": { "description": "Not Found", "schema": { "type": "object", "additionalProperties": { "type": "string" } } }
                }
            }
        },
        "/person/name/{name}": {
            "get": {
                "security": [{ "BearerAuth": [] }],
                "description": "İstenilen isme sahip kişiyi getirir",
                "produces": ["application/json"],
                "tags": ["persons"],
                "summary": "Belirli bir isme sahip kişiyi getir",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Kişi İsmi",
                        "name": "name",
                        "in": "path",
                        "required": true
                    },
                    {
                        "description": "User Name",
                        "name": "userName",
                        "in": "header",
                        "required": true,
                        "type": "string"
                    },
                    {
                        "description": "Refresh Token",
                        "name": "refreshToken",
                        "in": "header",
                        "required": false,
                        "type": "string"
                    }
                ],
                "responses": {
                    "200": { "description": "OK", "schema": { "$ref": "#/definitions/ServiceResponsePerson" } },
                    "404": { "description": "Not Found", "schema": { "type": "object", "additionalProperties": { "type": "string" } } }
                }
            }
        },
        "/persons": {
            "get": {
                "security": [{ "BearerAuth": [] }],
                "description": "Veritabanındaki tüm kişileri getirir",
                "produces": ["application/json"],
                "tags": ["persons"],
                "summary": "Tüm kişileri getir",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": { "$ref": "#/definitions/ServiceResponsePerson" }
                        }
                    }
                }
            },
            "post": {
                "security": [{ "BearerAuth": [] }],
                "description": "Birden fazla yeni kişi ekler",
                "consumes": ["application/json"],
                "produces": ["application/json"],
                "tags": ["persons"],
                "summary": "Birden fazla kişi ekle",
                "parameters": [
                    {
                        "description": "Birden fazla yeni kişi",
                        "name": "persons",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "type": "array",
                            "items": { "$ref": "#/definitions/Model.Person" }
                        }
                    },
                    {
                        "description": "User Name",
                        "name": "userName",
                        "in": "header",
                        "required": true,
                        "type": "string"
                    },
                    {
                        "description": "Refresh Token",
                        "name": "refreshToken",
                        "in": "header",
                        "required": false,
                        "type": "string"
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Created",
                        "schema": {
                            "type": "array",
                            "items": { "$ref": "#/definitions/ServiceResponsePerson" }
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": { "type": "object", "additionalProperties": { "type": "string" } }
                    }
                }
            }
        }
    },
    "definitions": {
        "Model.LoginModel": {
            "type": "object",
            "properties": {
                "password": { "type": "string" },
                "username": { "type": "string" }
            }
        },
        "Model.Person": {
            "type": "object",
            "properties": {
                "id": { "type": "integer" },
                "age": { "type": "integer" },
                "gender": { "type": "string" },
                "name": { "type": "string" },
                "username": { "type": "string" },
                "password": { "type": "string" }
            }
        },
		"ServiceResponsePerson": {
  			"type": "object",
  			"properties": {
    			"List": {
      				"type": "array",
      				"items": { "$ref": "#/definitions/Model.Person" }
    			},
    			"Entity": { "$ref": "#/definitions/Model.Person" },
				"Count": { "type": "integer" },
				"Token": { "type": "string" },
    			"RefreshToken": { "type": "string" },
    			"CreatedTokenTime": { "type": "string", "format": "date-time" },
    			"ValidationErrorList": {
      				"type": "array",
      				"items": { "type": "string" }
    			},
    			"Error": { "type": "string" }
  			}
		}
    }
}`

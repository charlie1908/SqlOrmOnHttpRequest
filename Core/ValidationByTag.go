package Core

import (
	"fmt"
	"github.com/patrickmn/go-cache"
	shared "httpRequestName/Shared"
	"reflect"
	"strings"
	"sync"
	"time"
)

//github.com/patrickmn/go-cache [Redis gibi otomatik silinen MameCache Mutex Locklar icin kullandim]
//https://chatgpt.com/share/67fef2a7-3dd4-8003-8890-cfe24fbf71f1

// https://chatgpt.com/share/67f39add-a20c-8003-8642-6b7f8d2f43d6
type Validator interface {
	Validate(reflect.Value, ...bool) (bool, error)
}

type EncryptValidator struct{}

func (e EncryptValidator) Validate(val reflect.Value, opts ...bool) (bool, error) {
	isRead := false
	if len(opts) > 0 {
		isRead = opts[0]
	}
	planeValue := val.Interface().(string)
	if isRead {
		decValue, err := Decrypt(planeValue, shared.Config.SECRETKEY)
		if err != nil {
			return false, fmt.Errorf("encryption failed: %w", err)
		}
		val.SetString(decValue)
	} else {
		encValue, err := Encrypt(planeValue, shared.Config.SECRETKEY)
		if err != nil {
			return false, fmt.Errorf("encryption failed: %w", err)
		}
		val.SetString(encValue)
	}

	return true, nil
}

type DecryptValidator struct{}

func (d DecryptValidator) Validate(val reflect.Value) (bool, error) {
	planeValue := val.Interface().(string)
	decValue, _ := Decrypt(planeValue, shared.Config.SECRETKEY)
	val.SetString(decValue)
	return true, nil
}

type HashValidator struct{}

func (h HashValidator) Validate(val reflect.Value, opts ...bool) (bool, error) {
	isRead := false
	if len(opts) > 0 {
		isRead = opts[0]
	}
	if !isRead {
		bytePlaneValue := []byte(val.Interface().(string))
		hash := HashAndSalt(bytePlaneValue)
		val.SetString(hash)
	}
	return true, nil
}

type DefaultValidator struct {
}

func (v DefaultValidator) Validate(val reflect.Value, _ ...bool) (bool, error) {
	return true, nil
}

func getValidatorFromTag(tag string) Validator {
	args := strings.Split(tag, ",")
	switch args[0] {
	case "hash":
		return HashValidator{}
	case "encrypt":
		return EncryptValidator{}
	}

	return DefaultValidator{}
}

const tagName = "validate"

func ValidateStruct(s interface{}, opts ...bool) []error {
	isRead := len(opts) > 0
	if len(opts) > 0 {
		isRead = opts[0]
	}
	var errs []error
	val := reflect.ValueOf(s)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
		for i := 0; i < val.Len(); i++ {
			item := val.Index(i)
			if item.Kind() == reflect.Ptr {
				item = item.Elem()
			}
			errs = append(errs, validateSingleStruct(item, isRead)...)
		}
	} else if val.Kind() == reflect.Struct {
		errs = append(errs, validateSingleStruct(val, isRead)...)
	}

	return errs
}

func validateSingleStruct(v reflect.Value, isRead bool) []error {
	var errs []error
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := v.Type().Field(i)
		tag := fieldType.Tag.Get(tagName)

		if tag == "" || tag == "-" {
			continue
		}

		validator := getValidatorFromTag(tag)
		var valid bool
		var err error
		if isRead {
			valid, err = validator.Validate(field, isRead)
		} else {
			valid, err = validator.Validate(field)
		}
		if !valid && err != nil {
			errs = append(errs, fmt.Errorf("%s %s", fieldType.Name, err.Error()))
		}
	}
	return errs
}

var mutexCache = cache.New(5*time.Minute, 10*time.Minute)

func GetUserMutex(userName string) *sync.Mutex {
	val, found := mutexCache.Get(userName)
	if found {
		return val.(*sync.Mutex)
	}

	mu := &sync.Mutex{}
	mutexCache.Set(userName, mu, cache.DefaultExpiration)
	return mu
}
func GetChangedFields(oldObj, newObj interface{}) map[string]interface{} {
	changes := make(map[string]interface{})

	oldVal := reflect.ValueOf(oldObj)
	newVal := reflect.ValueOf(newObj)

	if oldVal.Kind() == reflect.Ptr {
		oldVal = oldVal.Elem()
	}
	if newVal.Kind() == reflect.Ptr {
		newVal = newVal.Elem()
	}

	if oldVal.Kind() != reflect.Struct || newVal.Kind() != reflect.Struct {
		return changes
	}

	for i := 0; i < oldVal.NumField(); i++ {
		oldField := oldVal.Field(i)
		newField := newVal.Field(i)
		fieldType := oldVal.Type().Field(i)

		if !oldField.CanInterface() || !newField.CanInterface() {
			continue
		}

		oldValue := oldField.Interface()
		newValue := newField.Interface()

		switch newField.Kind() {
		case reflect.String:
			// Skip fields that have 'validate:"hash"' in their tag
			if newValue.(string) != "" && strings.Contains(fieldType.Tag.Get("validate"), "hash") {
				bytePass := []byte(strings.TrimSpace(newValue.(string)))
				if ComparePasswords(strings.TrimSpace(oldValue.(string)), bytePass) {
					continue
				} else {
					changes[fieldType.Name] = strings.TrimSpace(newValue.(string))
				}
			} else if newValue.(string) != "" && strings.TrimSpace(oldValue.(string)) != strings.TrimSpace(newValue.(string)) {
				changes[fieldType.Name] = strings.TrimSpace(newValue.(string))
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if oldValue != newValue && newField.Int() != 0 {
				changes[fieldType.Name] = newValue
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if oldValue != newValue && newField.Uint() != 0 {
				changes[fieldType.Name] = newValue
			}
		case reflect.Float32, reflect.Float64:
			if oldValue != newValue && newField.Float() != 0.0 {
				changes[fieldType.Name] = newValue
			}
		case reflect.Bool:
			if oldValue != newValue {
				changes[fieldType.Name] = newValue
			}
		}
	}
	return changes
}

/*var userLocks sync.Map // map[string]*sync.Mutex

func GetUserMutex(userName string) *sync.Mutex {
	val, _ := userLocks.LoadOrStore(userName, &sync.Mutex{})
	return val.(*sync.Mutex)
}*/

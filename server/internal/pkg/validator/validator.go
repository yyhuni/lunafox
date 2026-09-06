package validator

import (
	"reflect"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

var (
	uni      *ut.UniversalTranslator
	trans    ut.Translator
	validate *validator.Validate
)

// Init initializes the custom validator with translations
func Init() error {
	// Get the validator instance from Gin
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		validate = v

		// Use JSON tag name for JSON binding, form tag for query binding
		validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
			// Try json tag first
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "" || name == "-" {
				// Fall back to form tag
				name = strings.SplitN(fld.Tag.Get("form"), ",", 2)[0]
			}
			if name == "-" {
				return ""
			}
			return name
		})

		// Setup English translator
		enLocale := en.New()
		uni = ut.New(enLocale, enLocale)
		trans, _ = uni.GetTranslator("en")

		// Register default English translations
		if err := en_translations.RegisterDefaultTranslations(validate, trans); err != nil {
			return err
		}

		// Register custom translations for better messages
		registerCustomTranslations(validate, trans)
	}

	return nil
}

// registerCustomTranslations registers custom error messages
func registerCustomTranslations(v *validator.Validate, trans ut.Translator) {
	// Override min - handle both slice and number
	_ = v.RegisterTranslation("min", trans, func(ut ut.Translator) error {
		return ut.Add("min", "{0} must be at least {1}", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("min", fe.Field(), fe.Param())
		return t
	})

	// Override max - handle both slice and number
	_ = v.RegisterTranslation("max", trans, func(ut ut.Translator) error {
		return ut.Add("max", "{0} must be at most {1}", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("max", fe.Field(), fe.Param())
		return t
	})

	// Required
	_ = v.RegisterTranslation("required", trans, func(ut ut.Translator) error {
		return ut.Add("required", "{0} is required", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("required", fe.Field())
		return t
	})

	// oneof
	_ = v.RegisterTranslation("oneof", trans, func(ut ut.Translator) error {
		return ut.Add("oneof", "{0} must be one of [{1}]", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("oneof", fe.Field(), fe.Param())
		return t
	})
}

// TranslateError translates validation errors to human-readable messages
func TranslateError(err error) map[string]string {
	errs := make(map[string]string)

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			errs[e.Field()] = e.Translate(trans)
		}
	}

	return errs
}

// TranslateErrorToSlice translates validation errors to a slice of field errors
func TranslateErrorToSlice(err error) []FieldError {
	var errs []FieldError

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			errs = append(errs, FieldError{
				Field:   e.Field(),
				Message: e.Translate(trans),
			})
		}
	}

	return errs
}

// FieldError represents a single field validation error
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

package utils

// JWT payload should looks like:
//
// ```js
// {
// 	"k1": "v1",
// 	"k2": "v2",
// 	"k3": "v3",
// 	"uid": "laisky"
// }
// ```
//
// and the payload would be looks like:
//
// ```js
// {
// 	"expires_at": "2286-11-20T17:46:40Z",
// 	"k1": "v1",
// 	"k2": "v2",
// 	"k3": "v3",
// 	"uid": "laisky"
//   }
// ```

import (
	"fmt"
	"time"

	"github.com/Laisky/zap"

	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"

	jwt "github.com/dgrijalva/jwt-go"
)

var (
	// JWTSigningMethod default method to signing
	JWTSigningMethod = jwt.SigningMethodHS512
)

const (
	// JWTExpiresLayout default expires date stores in payload
	JWTExpiresLayout = time.RFC3339
	// JWTUserIDKey default key of user_id stores in token payload
	JWTUserIDKey = "uid"
	// JWTExpiresAtKey default key of expires_at stores in token payload
	JWTExpiresAtKey = "exp"
)

type baseJWT struct {
	JWTSigningMethod              *jwt.SigningMethodHMAC
	JWTUserIDKey, JWTExpiresAtKey string
}

// checkExpiresValid return the bool whether the `expires_at` is not expired
func (j *baseJWT) checkExpiresValid(now time.Time, expiresAtI interface{}) (ok bool, err error) {
	expiresAt, ok := expiresAtI.(string)
	if !ok {
		return false, fmt.Errorf("`%v` is not string", j.JWTExpiresAtKey)
	}
	tokenT, err := time.Parse(JWTExpiresLayout, expiresAt)
	if err != nil {
		return false, errors.Wrap(err, "try to parse token expires_at error")
	}

	return now.Before(tokenT), nil
}

// JWT struct to generate and validate jwt tokens
//
// use a global uniform secret to signing all token.
type JWT struct {
	*JwtCfg
}

// JwtCfg configuration of JWT
type JwtCfg struct {
	baseJWT
	Secret []byte
}

// NewJWTCfg create new JwtCfg  with secret
func NewJWTCfg(secret []byte) *JwtCfg {
	return &JwtCfg{
		Secret: secret,
		baseJWT: baseJWT{
			JWTSigningMethod: JWTSigningMethod,
			JWTUserIDKey:     JWTUserIDKey,
			JWTExpiresAtKey:  JWTExpiresAtKey,
		},
	}
}

// NewJWT create new JWT with JwtCfg
func NewJWT(cfg *JwtCfg) (*JWT, error) {
	if len(cfg.Secret) == 0 {
		return nil, errors.New("jwtCfg.Secret should not be empty")
	}

	jwt.TimeFunc = Clock.GetUTCNow

	return &JWT{
		JwtCfg: cfg,
	}, nil
}

// // Setup (deprecated) initialize JWT
// func (j *JWT) Setup(secret string) {
// 	// const key names
// 	j.JWTExpiresAtKey = "expires_at"
// 	j.JWTUserIDKey = "uid"

// 	j.Secret = []byte(secret)
// }

// Generate (Deprecated) generate JWT token.
// old interface
func (j *JWT) Generate(expiresAt int64, payload map[string]interface{}) (string, error) {
	jwtPayload := jwt.MapClaims{}
	for k, v := range payload {
		jwtPayload[k] = v
	}
	jwtPayload["expires_at"] = ParseTs2String(expiresAt, JWTExpiresLayout)

	token := jwt.NewWithClaims(JWTSigningMethod, jwtPayload)
	tokenStr, err := token.SignedString(j.Secret)
	if err != nil {
		return "", errors.Wrap(err, "try to signed token got error")
	}
	return tokenStr, nil
}

// // GenerateToken generate JWT token.
// // do not use `expires_at` & `uid` as keys.
// func (j *JWT) GenerateToken(userID string, expiresAt time.Time, payload map[string]interface{}) (tokenStr string, err error) {
// 	jwtPayload := jwt.MapClaims{}
// 	for k, v := range payload {
// 		jwtPayload[k] = v
// 	}
// 	jwtPayload[j.JWTExpiresAtKey] = expiresAt.Unix()
// 	jwtPayload[j.JWTUserIDKey] = userID

// 	token := jwt.NewWithClaims(JWTSigningMethod, jwtPayload)
// 	if tokenStr, err = token.SignedString(j.Secret); err != nil {
// 		return "", errors.Wrap(err, "try to signed token got error")
// 	}
// 	return tokenStr, nil
// }

// GenerateToken generate JWT token with userID(interface{})
func (j *JWT) GenerateToken(userID interface{}, expiresAt time.Time, payload map[string]interface{}) (tokenStr string, err error) {
	jwtPayload := jwt.MapClaims{}
	for k, v := range payload {
		jwtPayload[k] = v
	}
	jwtPayload[j.JWTExpiresAtKey] = expiresAt.Unix()
	jwtPayload[j.JWTUserIDKey] = userID

	token := jwt.NewWithClaims(JWTSigningMethod, jwtPayload)
	if tokenStr, err = token.SignedString(j.Secret); err != nil {
		return "", errors.Wrap(err, "try to signed token got error")
	}
	return tokenStr, nil
}

// // Validate (deprecated) validate the token and return the payload
// //
// // if token is invalidate, err will not be nil.
// func (j *JWT) Validate(tokenStr string) (payload map[string]interface{}, err error) {
// 	Logger.Debug("Validate for token", zap.String("tokenStr", tokenStr))
// 	var (
// 		claims jwt.MapClaims
// 		ok     bool
// 	)
// 	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
// 		// Don't forget to validate the alg is what you expect:
// 		if method, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || method != j.JWTSigningMethod {
// 			return nil, errors.New("JWT method not allowd")
// 		}
// 		return j.Secret, nil
// 	})
// 	if err != nil || !token.Valid {
// 		// delay return after got payload
// 		err = errors.Wrap(err, "token invalidate")
// 	}

// 	payload = map[string]interface{}{}
// 	if claims, ok = token.Claims.(jwt.MapClaims); !ok {
// 		return nil, errors.New("payload type not match `map[string]interface{}`")
// 	}
// 	for k, v := range claims {
// 		payload[k] = v
// 	}
// 	if err != nil { // no need for furthur validate
// 		return payload, err
// 	}

// 	if !claims.VerifyExpiresAt(Clock.GetUTCNow().Unix(), true) {
// 		return payload, fmt.Errorf("token expired")
// 	}
// 	if _, ok = payload[j.JWTUserIDKey]; !ok {
// 		return payload, fmt.Errorf("token must contains `%v`", j.JWTUserIDKey)
// 	}

// 	return payload, err
// }

// Validate validate the token and return the payload
//
// if token is invalidate, err will not be nil.
func (j *JWT) Validate(tokenStr string) (payload jwt.MapClaims, err error) {
	Logger.Debug("Validate for token", zap.String("tokenStr", tokenStr))
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if method, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || method != j.JWTSigningMethod {
			return nil, errors.New("JWT method not allowd")
		}
		return j.Secret, nil
	})
	if err != nil || !token.Valid {
		// return after got payload
		err = errors.Wrap(err, "token invalidate")
	}

	var ok bool
	if payload, ok = token.Claims.(jwt.MapClaims); !ok {
		return nil, errors.New("payload type not match `map[string]interface{}`")
	}
	if err != nil {
		return payload, err
	}

	if !payload.VerifyExpiresAt(Clock.GetUTCNow().Unix(), true) { // exp must exists
		return payload, fmt.Errorf("token expired")
	}
	if _, ok = payload[j.JWTUserIDKey]; !ok {
		err = fmt.Errorf("token must contains `%v`", j.JWTUserIDKey)
	}
	return payload, err
}

// GeneratePasswordHash generate hashed password by origin password
func GeneratePasswordHash(password []byte) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

// ValidatePasswordHash validate password is match with hashedPassword
func ValidatePasswordHash(hashedPassword, password []byte) bool {
	return bcrypt.CompareHashAndPassword(hashedPassword, password) == nil
}

// DivideJWT jwt utils to generate and validate token.
//
// use seperate secret for each token
type DivideJWT struct {
	*DivideJWTCfg
}

// JWTUserModel load secret by uid
type JWTUserModel interface {
	GetUID() interface{}
	LoadSecretByUID(uid interface{}) ([]byte, error)
}

// DivideJWTCfg configuration
type DivideJWTCfg struct {
	baseJWT
}

// NewDivideJWTCfg create new JwtCfg  with secret
func NewDivideJWTCfg() *DivideJWTCfg {
	jwt.TimeFunc = Clock.GetUTCNow
	return &DivideJWTCfg{
		baseJWT: baseJWT{
			JWTSigningMethod: JWTSigningMethod,
			JWTUserIDKey:     JWTUserIDKey,
			JWTExpiresAtKey:  JWTExpiresAtKey,
		},
	}
}

// NewDivideJWT create new JWT with JwtCfg
func NewDivideJWT(cfg *DivideJWTCfg) (*DivideJWT, error) {
	if cfg.JWTUserIDKey == "" ||
		cfg.JWTExpiresAtKey == "" ||
		cfg.JWTSigningMethod == nil {
		return nil, fmt.Errorf("configuration error")
	}

	return &DivideJWT{
		DivideJWTCfg: cfg,
	}, nil
}

// GenerateToken generate JWT token.
// do not use `expires_at` & `uid` as keys.
func (j *DivideJWT) GenerateToken(user JWTUserModel, expiresAt time.Time, payload map[string]interface{}) (tokenStr string, err error) {
	jwtPayload := jwt.MapClaims{}
	for k, v := range payload {
		jwtPayload[k] = v
	}
	jwtPayload[j.JWTExpiresAtKey] = expiresAt.Unix()
	jwtPayload[j.JWTUserIDKey] = user.GetUID()

	token := jwt.NewWithClaims(JWTSigningMethod, jwtPayload)
	var secret []byte
	if secret, err = user.LoadSecretByUID(user.GetUID()); err != nil {
		Logger.Error("try to load jwt secret by uid got error",
			zap.Error(err),
			zap.String("uid", fmt.Sprint(user.GetUID())))
		return "", err
	}
	if tokenStr, err = token.SignedString(secret); err != nil {
		return "", errors.Wrap(err, "try to signed token got error")
	}
	return tokenStr, nil
}

// Validate validate the token and return the payload
//
// if token is invalidate, err will not be nil.
func (j *DivideJWT) Validate(user JWTUserModel, tokenStr string) (payload jwt.MapClaims, err error) {
	Logger.Debug("Validate for token", zap.String("tokenStr", tokenStr))
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if method, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || method != j.JWTSigningMethod {
			return nil, errors.New("JWT method not allowd")
		}
		return user.LoadSecretByUID(user.GetUID())
	})
	if err != nil || !token.Valid {
		// return after got payload
		err = errors.Wrap(err, "token invalidate")
	}

	var ok bool
	if payload, ok = token.Claims.(jwt.MapClaims); !ok {
		return nil, errors.New("payload type not match `map[string]interface{}`")
	}
	if err != nil {
		return payload, err
	}

	if !payload.VerifyExpiresAt(Clock.GetUTCNow().Unix(), true) { // exp must exists
		return payload, fmt.Errorf("token expired")
	}
	if _, ok = payload[j.JWTUserIDKey]; !ok {
		err = fmt.Errorf("token must contains `%v`", j.JWTUserIDKey)
	}
	return payload, err
}

package token

import (
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	// "github.com/google/uuid"
)

const TEMPSECRET = "cbb3589b-8035-4ac2-b978-23dd0d0052b5-4d2f599e-0ff7-476c-a7ed-de66b64244f6"

// Token: a JWT Token creator struct. For both User credentials
//        and for Chatroom Membership credentials
type Token struct {
  Token string  `codec:"token"`
}

func CreateToken(uid string)( *Token, error ){
  token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
    "token_id": uid,
    "exp":     time.Now().Add(time.Hour * 72).Unix(),
  })

  tokenString, err := token.SignedString([]byte(TEMPSECRET))
  if err != nil {
    log.Printf(" -> ERROR: CreateToken: Failed to Sign Token")
    return nil, err
  }

  userToken := Token{
    Token: tokenString,
  }

  return &userToken, nil
}

func CreateTokenNoExpiration(uid string)( *Token, error ){
  token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
    "token_id": uid,
  })

  tokenString, err := token.SignedString([]byte(TEMPSECRET))
  if err != nil {
    log.Printf(" -> ERROR: CreateToken: Failed to Sign Token")
    return nil, err
  }

  userToken := Token{
    Token: tokenString,
  }

  return &userToken, nil
}

func( userToken *Token )parseToken()( *jwt.Token, error ){
  token, err := jwt.Parse(
    userToken.Token,
    func(token *jwt.Token)( interface{},error ){
      if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
        return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
      }
      return []byte(TEMPSECRET), nil
  })

  return token, err
}

func( userToken *Token )GetClaims(
)( jwt.MapClaims, error ){
  token, err := userToken.parseToken()
  if err != nil {
    log.Printf(" -> Error: Failed to Parse Token")
    return nil, err
  }

  if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
    return claims, nil
  }
  return nil, fmt.Errorf("Failed to retreive claims")
}

func(userToken *Token)Validate() error {
  // First, we Validate that the token is at least a valid JWT Token
  token, err := jwt.Parse(userToken.Token, func(token *jwt.Token) (interface{}, error) {
    // Don't forget to validate the alg is what you expect:
    if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
      return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
    }

    // hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
    return []byte(TEMPSECRET), nil
  })
  if err != nil {
    log.Printf(" -> Error: Failed to Valdate Token")
    return err
  }

  if _, ok := token.Claims.(jwt.MapClaims); !ok && !token.Valid {
    return fmt.Errorf(" -> Error: Failed to Validate Token")
  }

  return nil
}

func( userToken *Token )TokenIsExpired()( bool,error ){
  claims, err := userToken.GetClaims()
  if err != nil {
    return false, err
  }
  now := time.Now().Unix()
  exp, err := claims.GetExpirationTime()
  if err != nil {
    log.Printf(" -> Error: Unknown eroor occurred while retreiving expiration from Claims")
    return false, err
  }
  return exp.Unix() >= now, nil
}

func( userToken *Token )GetTokenID()( string,error ){
  claims, err := userToken.GetClaims()
  if err != nil {
    return "", err
  }
  uid := claims["token_id"].(string)
  return uid, nil
}

func( userToken *Token )RefreshToken()( *Token,error ){
  claims, err := userToken.GetClaims()
  if err != nil {
    return nil, err
  }
  uid := claims["token_id"].(string)

  newToken, err := CreateToken(uid)
  if err != nil {
    return nil, err
  }
  userToken = newToken

  return newToken, nil
}

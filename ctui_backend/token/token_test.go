package token

import (
	"fmt"
	"testing"
)

func TestToken(t *testing.T) {
  t.Run("Create-Token and Authenticate", func(t *testing.T){
    userToken, err := CreateToken("UserId1234")
    if err != nil {
      t.Errorf("FAILED: Failed to create Token: %v", err.Error())
      return
    }

    err = userToken.Validate()
    if err != nil {
      t.Errorf("FAILED: Failed to Authenticate Token: %v", err.Error())
      return
    }
    fmt.Printf("PASSED: Got Token\n")
  })
  t.Run("Get UserID", func(t *testing.T){
    userID := "USERID1234"
    userToken, err := CreateToken(userID)
    if err != nil {
      t.Errorf("FAILED: Failed to create Token: %v", err.Error())
      return
    }

    retreivedID, err := userToken.GetUserID()
    if err != nil {
      t.Errorf("FAILED: Failed to retreive UserID: %v", err.Error())
      return
    }

    if userID != retreivedID {
      t.Errorf("FAILED: Got %v Want %v", retreivedID, userID)
      return
    }
    fmt.Printf("Get UserID: PASSED \n")
  })
}

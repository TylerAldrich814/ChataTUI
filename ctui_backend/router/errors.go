package router

import "fmt"

// ----------------------- Router Errors --------------------------

type InvalidChatroomError struct{ cm string }
func(e InvalidChatroomError)Error() string {
  return fmt.Sprintf("Error: databaseError - Chatroom \"%s\" doesn't exist", e.cm)
}

type InvalidTokenError struct{}
func(e InvalidTokenError)Error() string {
  return "Error: tokenError - Token providerd is invalid"
}

type InvalidTokenIDError struct{}
func(e InvalidTokenIDError)Error() string {
  return "Error: tokenError - Could not extract TokenID"
}

type InvalidUserIDError struct{ id string }
func(e InvalidUserIDError)Error() string {
  return fmt.Sprintf("Error: databaseError - Invalid UsedID \"%s\"", e.id)
}

type InvalidUUIDError struct{}
func(e InvalidUUIDError)Error() string {
  return "Error: authenticationError - TokenID is not a valid UUID"
}

type TokenIsExpiredError struct{}
func(e TokenIsExpiredError)Error() string {
  return "Error: tokenError - Token beyond it's expiration date."
}
type TokenNotFoundError struct{}
func(e TokenNotFoundError)Error() string {
  return "Error: tokenError - Token provided is not in the database"
}

type MalformedTokenError struct{}
func(e MalformedTokenError)Error() string {
  return "Error: tokenError - Malformed Token"
}

type MissingAuthHeaderError struct{}
func(e MissingAuthHeaderError)Error() string {
  return "Error: authenticationError - Authentication Header is missing"
}

type TokenExtractionError struct{}
func(e TokenExtractionError) Error() string {
  return "Error: authenticationError - Couldn't extract TokenID"
}

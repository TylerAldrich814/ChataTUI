package db

import "fmt"

// ----------------------- Database Errors --------------------------

type BucketNotFoundError struct{ d string }
func(e BucketNotFoundError)Error() string {
  return fmt.Sprintf("Error: DatabaseError - Bucket \"%s\" not found:  ", e.d)
}

// DataNotFoundError: d: string = DataKey b: string = bucket
type DataNotFoundError struct{ d, b string }
func(e DataNotFoundError)Error() string {
  return fmt.Sprintf(
    "Error: DatabaseError - Could not fetch \"%s\" from Bucket \"%s\"",
    e.d, e.b,
  )
}

type EncoderError struct { err string }
func(e EncoderError)Error() string {
  return fmt.Sprintf("Error: DatabaseError - Failed to Create Encoder: %s", e.err)
}

type DecoderError struct { err string }
func(e DecoderError)Error() string {
  return fmt.Sprintf("Error: DatabaseError - Failed to Create Decoder: %s", e.err)
}

type GetDataError struct{ i, b string }
func(e GetDataError)Error() string {
  return fmt.Sprintf(
    "Error: DatabaseError - Failure to Get item \"%s\" in Bucket \"%s\"",
    e.i,e.b,
  )
}
type PutDataError struct{ i, b, e string }
func(e PutDataError)Error() string {
  return fmt.Sprintf(
    "Error: DatabaseError - Failure to Put item \"%s\" in Bucket \"%s\" :: %s",
    e.i,e.b, e.e,
  )
}

type DeleteDataError struct{ i, b, e string }
func(e DeleteDataError)Error() string {
  return fmt.Sprintf(
    "Error: DatabaseError - Failure to Delete item \"%s\" in Bucket \"%s\" :: %s",
    e.i,e.b, e.e,
  )
}

type FailedSecurityCheckError struct{ t, err string }
func(e FailedSecurityCheckError)Error() string {
  return fmt.Sprintf(
    "Error: DatabaseError - Security Authorization for %s Failed: %s",
    e.t, e.err,
  )
}

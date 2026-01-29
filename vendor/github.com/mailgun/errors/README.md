# Errors
An error handling package to add additional structured fields to errors. This package helps you keep the
[only handle errors once rule](https://dave.cheney.net/2016/04/27/dont-just-check-errors-handle-them-gracefully)
while not losing context where the error occurred.

## Usage
#### errors.Wrap()
includes a stack trace so logging can report the exact location where the error occurred. 
*Includes `Wrapf()` and `Wrap()` variants*
```go
return errors.Wrapf(err, "while reading '%s'", fileName)
```
#### errors.Stack()
Identical to `errors.Wrap()` but you don't need a message, just a stack trace to where the error occurred.
```go
return errors.Stack(err)
```
#### errors.Fields{}
Attach additional fields to the error and a stack trace to give structured logging as much context
to the error as possible. *Includes `Wrap()`, `Wrapf()`, `Stack()`, `Error()` and `Errorf()` variants*
```go
return errors.Fields{"fileName": fileName}.Wrapf(err, "while reading '%s'", fileName)
return errors.Fields{"fileName": fileName}.Stack(err)
return errors.Fields{"fileName": fileName}.Error("while reading")
```
#### errors.WrapFields()
Works just like `errors.Fields{}` but allows collecting and passing around fields independent of the point of error 
creation. In functions with many exit points this can result in cleaner less cluttered looking code.
```go
fields := map[string]any{
    "domain.id": domainId,
}
err, accountID := account.GetByDomain(domainID)
if err != nil {
    // Error only includes `domain.id`
    return errors.WrapFields(err, fields, "during call to account.GetByDomain()")
}
fields["account.id"] = accountID

err, disabled := domain.Disable(accountID, domainID)
if err != nil {
    // Error now includes `account.id` and `domain.id`
    return errors.WrapFields(err, fields, "during call to domain.Disable()")
}
```
#### errors.Last()
Works just like `errors.As()` except it returns the last error in the chain instead of the first. In
this way you can discover the target which is closest to where the error occurred.
```go
// Returns the last error in the chain that has a stack trace attached
var last callstack.HasStackTrace
if errors.Last(err, &last)) {
	fmt.Printf("Error occurred here: %+v", last.StackTrace())
}
```
#### errors.ToMap()
A convenience function to extract all stack and field information from the error.
```go
err := io.EOF
err = errors.WithFields{"fileName": "file.txt"}.Wrap(err, "while reading")
m := errors.ToMap(err)
fmt.Printf("%#v\n", m)
// OUTPUT
// map[string]interface {}{
//   "excFileName":"/path/to/wrap_test.go",
//   "excFuncName":"my_package.ReadAFile",
//   "excLineNum":42,
//   "excType":"*errors.errorString",
//   "excValue":"while reading: EOF",
//   "fileName":"file.txt"
//  }
```
#### errors.ToLogrus()
A convenience function to extract all stack and field information from the error in a form
appropriate for logrus.
```go
err := io.EOF
err = errors.WithFields{"fileName": "file.txt"}.Wrap(err, "while reading")
f := errors.ToLogrus(err)
logrus.WithFields(f).Info("test logrus fields")
// OUTPUT
// time="2023-02-20T19:11:05-06:00"
//   level=info
//   msg="test logrus fields"
//   excFileName=/path/to/wrap_test.go
//   excFuncName=my_package.ReadAFile
//   excLineNum=21
//   excType="*errors.wrappedError"
//   excValue="while reading: EOF"
```

## Convenience to std error library methods
Provides pass through access to the standard `errors.Is()`, `errors.As()`, `errors.Unwrap()` so you don't need to
import this package and the standard error package.

## Supported by internal tooling
If you are working at mailgun and are using scaffold; using `logrus.WithError(err)` will cause logrus to 
automatically retrieve the fields attached to the error and index them into our logging system as separate
searchable fields.

## Perfect for passing additional information to http handler middleware
If you have custom http middleware for handling unhandled errors, this is an excellent way
to easily pass additional information about the request up to the error handling middleware.

## Support for standard golang introspection functions
Errors wrapped with `errors.WithFields{}` are compatible with standard library introspection functions `errors.Unwrap()`,
`errors.Is()` and `errors.As()`
```go
ErrQuery := errors.New("query error")
wrap := errors.WithFields{"key1": "value1"}.Wrap(err, "message")
errors.Is(wrap, ErrQuery) // == true
```

## Proper Usage
The fields wrapped by `errors.WithFields{}` are not intended to be used to by code to decide how an error should be 
handled. It is intended as a convenience where the failure is well known, but the context is dynamic. In other words,
you know the database returned an unrecoverable query error, but you want to attach localized context information
to the error.

As an example
```go
func (r *Repository) FetchAuthor(customerID, isbn string) (Author, error) {
    // Returns ErrorNotFound{} if not exist
    book, err := r.fetchBook(isbn)
    if err != nil {
        return nil, errors.WithFields{"customer.id": customerID, "isbn": isbn}.Wrap(err, "while fetching book")
    }
    // Returns ErrorNotFound{} if not exist
    author, err := r.fetchAuthorByBook(book)
    if err != nil {
        return nil, errors.WithFields{"customer.id" customerID, "book": book}.Wrap(err, "while fetching author")
    }
    return author, nil
}
```
Now you can easily search your structured logs for errors related to `customer.id`.

You should continue to create and inspect custom error types
```go

type ErrAuthorNotFound struct {
    Msg string
}

func (e *ErrAuthorNotFound) Error() string {
    return e.Msg
}

func (e *ErrAuthorNotFound) Is(target error) bool {
    _, ok := target.(*NotFoundError)
    return ok
}

func main() {
    r := Repository{}
    author, err := r.FetchAuthor("isbn-213f-23422f52356")
    if err != nil {
        // Fetch the original and determine if the error is recoverable
        if error.Is(err, &ErrAuthorNotFound{}) {
            author, err := r.AddBook("isbn-213f-23422f52356", "charles", "darwin")
        }
        if err != nil {
            logrus.WithFields(errors.ToLogrus(err)).
				WithError(err).Error("while fetching author")
            os.Exit(1)
        }
    }
    fmt.Printf("Author %+v\n", author)
}
```

package viber

// Error from Viber
type Error struct {
	Status        int
	StatusMessage string
}

// Error interface function
func (e Error) Error() string {
	//return fmt.Sprintf("Viber error, status ID: %d Status: %s", id, status)
	return e.StatusMessage
}

// ErrorStatus code of Viber error, returns -1 if e is not Viber error
func ErrorStatus(e interface{}) int {
	switch e.(type) {
	case Error:
		return e.(Error).Status
	}
	return -1
}

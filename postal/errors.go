package postal

type CCDownError struct {
    message string
}

func (err CCDownError) Error() string {
    return err.message
}

type UAADownError struct {
    message string
}

func (err UAADownError) Error() string {
    return err.message
}

type UAAUserNotFoundError struct {
    message string
}

func (err UAAUserNotFoundError) Error() string {
    return err.message
}

type UAAGenericError struct {
    message string
}

func (err UAAGenericError) Error() string {
    return err.message
}

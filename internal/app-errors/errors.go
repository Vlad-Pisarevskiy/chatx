package app_errors

import "errors"

var ErrLoginNotFound = errors.New("login is not found in DB")

var ErrInvalidToken = errors.New("token does not exist")

var ErrEmptyLogin = errors.New("login is empty")

var ErrShortLogin = errors.New("login is too short")

var ErrLongLogin = errors.New("login is too long")

var ErrEmptyPassword = errors.New("password is empty")

var ErrShortPassword = errors.New("password is too short")

var ErrLongPassword = errors.New("password is too long")

var ErrEmptyName = errors.New("name is empty")

var ErrLongName = errors.New("name is too long")

var ErrExistsLogin = errors.New("login is already exists")

var ErrIncorrectLoginData = errors.New("login or password is incorrect")

var ErrEmptyDatabasePath = errors.New("empty database path")

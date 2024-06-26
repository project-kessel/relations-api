// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: health/v1/health.proto

package v1

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/anypb"
)

// ensure the imports are used
var (
	_ = bytes.MinRead
	_ = errors.New("")
	_ = fmt.Print
	_ = utf8.UTFMax
	_ = (*regexp.Regexp)(nil)
	_ = (*strings.Reader)(nil)
	_ = net.IPv4len
	_ = time.Duration(0)
	_ = (*url.URL)(nil)
	_ = (*mail.Address)(nil)
	_ = anypb.Any{}
	_ = sort.Sort
)

// Validate checks the field values on GetLivezRequest with the rules defined
// in the proto definition for this message. If any rules are violated, the
// first error encountered is returned, or nil if there are no violations.
func (m *GetLivezRequest) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on GetLivezRequest with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// GetLivezRequestMultiError, or nil if none found.
func (m *GetLivezRequest) ValidateAll() error {
	return m.validate(true)
}

func (m *GetLivezRequest) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if len(errors) > 0 {
		return GetLivezRequestMultiError(errors)
	}

	return nil
}

// GetLivezRequestMultiError is an error wrapping multiple validation errors
// returned by GetLivezRequest.ValidateAll() if the designated constraints
// aren't met.
type GetLivezRequestMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m GetLivezRequestMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m GetLivezRequestMultiError) AllErrors() []error { return m }

// GetLivezRequestValidationError is the validation error returned by
// GetLivezRequest.Validate if the designated constraints aren't met.
type GetLivezRequestValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e GetLivezRequestValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e GetLivezRequestValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e GetLivezRequestValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e GetLivezRequestValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e GetLivezRequestValidationError) ErrorName() string { return "GetLivezRequestValidationError" }

// Error satisfies the builtin error interface
func (e GetLivezRequestValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sGetLivezRequest.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = GetLivezRequestValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = GetLivezRequestValidationError{}

// Validate checks the field values on GetLivezReply with the rules defined in
// the proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *GetLivezReply) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on GetLivezReply with the rules defined
// in the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in GetLivezReplyMultiError, or
// nil if none found.
func (m *GetLivezReply) ValidateAll() error {
	return m.validate(true)
}

func (m *GetLivezReply) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for Status

	// no validation rules for Code

	if len(errors) > 0 {
		return GetLivezReplyMultiError(errors)
	}

	return nil
}

// GetLivezReplyMultiError is an error wrapping multiple validation errors
// returned by GetLivezReply.ValidateAll() if the designated constraints
// aren't met.
type GetLivezReplyMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m GetLivezReplyMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m GetLivezReplyMultiError) AllErrors() []error { return m }

// GetLivezReplyValidationError is the validation error returned by
// GetLivezReply.Validate if the designated constraints aren't met.
type GetLivezReplyValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e GetLivezReplyValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e GetLivezReplyValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e GetLivezReplyValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e GetLivezReplyValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e GetLivezReplyValidationError) ErrorName() string { return "GetLivezReplyValidationError" }

// Error satisfies the builtin error interface
func (e GetLivezReplyValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sGetLivezReply.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = GetLivezReplyValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = GetLivezReplyValidationError{}

// Validate checks the field values on GetReadyzRequest with the rules defined
// in the proto definition for this message. If any rules are violated, the
// first error encountered is returned, or nil if there are no violations.
func (m *GetReadyzRequest) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on GetReadyzRequest with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// GetReadyzRequestMultiError, or nil if none found.
func (m *GetReadyzRequest) ValidateAll() error {
	return m.validate(true)
}

func (m *GetReadyzRequest) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if len(errors) > 0 {
		return GetReadyzRequestMultiError(errors)
	}

	return nil
}

// GetReadyzRequestMultiError is an error wrapping multiple validation errors
// returned by GetReadyzRequest.ValidateAll() if the designated constraints
// aren't met.
type GetReadyzRequestMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m GetReadyzRequestMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m GetReadyzRequestMultiError) AllErrors() []error { return m }

// GetReadyzRequestValidationError is the validation error returned by
// GetReadyzRequest.Validate if the designated constraints aren't met.
type GetReadyzRequestValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e GetReadyzRequestValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e GetReadyzRequestValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e GetReadyzRequestValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e GetReadyzRequestValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e GetReadyzRequestValidationError) ErrorName() string { return "GetReadyzRequestValidationError" }

// Error satisfies the builtin error interface
func (e GetReadyzRequestValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sGetReadyzRequest.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = GetReadyzRequestValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = GetReadyzRequestValidationError{}

// Validate checks the field values on GetReadyzReply with the rules defined in
// the proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *GetReadyzReply) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on GetReadyzReply with the rules defined
// in the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in GetReadyzReplyMultiError,
// or nil if none found.
func (m *GetReadyzReply) ValidateAll() error {
	return m.validate(true)
}

func (m *GetReadyzReply) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for Status

	// no validation rules for Code

	if len(errors) > 0 {
		return GetReadyzReplyMultiError(errors)
	}

	return nil
}

// GetReadyzReplyMultiError is an error wrapping multiple validation errors
// returned by GetReadyzReply.ValidateAll() if the designated constraints
// aren't met.
type GetReadyzReplyMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m GetReadyzReplyMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m GetReadyzReplyMultiError) AllErrors() []error { return m }

// GetReadyzReplyValidationError is the validation error returned by
// GetReadyzReply.Validate if the designated constraints aren't met.
type GetReadyzReplyValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e GetReadyzReplyValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e GetReadyzReplyValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e GetReadyzReplyValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e GetReadyzReplyValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e GetReadyzReplyValidationError) ErrorName() string { return "GetReadyzReplyValidationError" }

// Error satisfies the builtin error interface
func (e GetReadyzReplyValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sGetReadyzReply.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = GetReadyzReplyValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = GetReadyzReplyValidationError{}

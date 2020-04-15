package v1alpha1

type PlayRunErrorMessage string

const (
	// NoMoreFrames No more frames to play
	NoMoreFrames PlayRunErrorMessage = "No more frames to play"
)

// PlayRunError represents an error during the run
type PlayRunError struct {
	Message PlayRunErrorMessage
}

var _ error = &PlayRunError{}

func (e *PlayRunError) Error() string {
	return string(e.Message)
}

func NewError(m PlayRunErrorMessage) *PlayRunError {
	return &PlayRunError{m}
}

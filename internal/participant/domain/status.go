package domain

// Status is the registration lifecycle. A participant is born pending and
// becomes confirmed at the single confirmation point — today immediately
// (free races), tomorrow after a payment succeeds.
type Status string

const (
	StatusPending   Status = "pendiente"
	StatusConfirmed Status = "confirmado"
)

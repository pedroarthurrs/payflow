package domain

import "context"

// TxManager abstracts database transaction management.
type TxManager interface {
	// WithTx executes fn inside a transaction. If fn returns an error, the
	// transaction is rolled back; otherwise it is committed.
	WithTx(ctx context.Context, fn func(tx TxHandle) error) error
}

// TxHandle is passed to transaction callbacks, providing scoped repositories.
type TxHandle interface {
	CustomerRepo() CustomerRepository
	AddressRepo() AddressRepository
	CredentialRepo() CredentialRepository
	TransactionFSMRepo() TransactionFSMRepository
}

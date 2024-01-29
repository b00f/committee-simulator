package sandbox

import (
	"github.com/pactus-project/pactus/committee"
	"github.com/pactus-project/pactus/crypto"
	"github.com/pactus-project/pactus/crypto/bls"
	"github.com/pactus-project/pactus/sandbox"
	"github.com/pactus-project/pactus/sortition"
	"github.com/pactus-project/pactus/types/account"
	"github.com/pactus-project/pactus/types/param"
	"github.com/pactus-project/pactus/types/tx"
	"github.com/pactus-project/pactus/types/validator"
)

func hasDuplicates(vector []*validator.Validator) bool {
	countMap := make(map[int32]int)

	for _, val := range vector {
		countMap[val.Number()]++
		if countMap[val.Number()] > 1 {
			return true
		}
	}

	return false
}

var _ sandbox.Sandbox = &MockSandbox{}

// MockSandbox is a testing mock for sandbox.
type MockSandbox struct {
	MockCommittee    committee.Committee
	MockValidators   map[crypto.Address]*validator.Validator
	MockParams       param.Params
	CurHeight        uint32
	JoinedValidators []*validator.Validator
}

func (m *MockSandbox) Account(addr crypto.Address) *account.Account {
	panic("unreachable")
}
func (m *MockSandbox) MakeNewAccount(_ crypto.Address) *account.Account {
	panic("unreachable")
}
func (m *MockSandbox) UpdateAccount(addr crypto.Address, acc *account.Account) {
	panic("unreachable")
}
func (m *MockSandbox) Validator(addr crypto.Address) *validator.Validator {
	val := m.MockValidators[addr]
	return val
}
func (m *MockSandbox) JoinedToCommittee(addr crypto.Address) {
	val := m.MockValidators[addr]
	m.JoinedValidators = append(m.JoinedValidators, val)

	if hasDuplicates(m.JoinedValidators) {
		panic("has duplicated")
	}
}
func (m *MockSandbox) IsJoinedCommittee(addr crypto.Address) bool {
	panic("unreachable")
}
func (m *MockSandbox) MakeNewValidator(pub *bls.PublicKey) *validator.Validator {
	panic("unreachable")
}
func (m *MockSandbox) UpdateValidator(val *validator.Validator) {
	m.MockValidators[val.Address()] = val
}
func (m *MockSandbox) CurrentHeight() uint32 {
	return m.CurHeight
}
func (m *MockSandbox) Params() *param.Params {
	return &m.MockParams
}
func (m *MockSandbox) IterateAccounts(consumer func(crypto.Address, *account.Account, bool)) {
	panic("unreachable")
}
func (m *MockSandbox) IterateValidators(consumer func(*validator.Validator, bool, bool)) {
	for _, val := range m.JoinedValidators {
		consumer(val, false, true)
	}
}

func (m *MockSandbox) Committee() committee.Reader {
	return m.MockCommittee
}

func (m *MockSandbox) VerifyProof(uint32, sortition.Proof, *validator.Validator) bool {
	return true
}

func (m *MockSandbox) Reset() {
	m.JoinedValidators = make([]*validator.Validator, 0, 10)
}

func (m *MockSandbox) UpdatePowerDelta(delta int64) {
	panic("unreachable")
}

func (m *MockSandbox) PowerDelta() int64 {
	panic("unreachable")
}

func (m *MockSandbox) AccumulatedFee() int64 {
	return 0
}

func (m *MockSandbox) CommitTransaction(trx *tx.Tx) {

}

func (m *MockSandbox) AnyRecentTransaction(txID tx.ID) bool {
	return false
}

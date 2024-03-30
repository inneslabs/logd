package sign

type Signer interface {
	Sign(secret, payload []byte) ([]byte, error)
	Verify(secret, pkg *Pkg) (bool, error)
}

package service

// LoginCodeService mints and checks the one-time codes used for passwordless
// sign-in.
//
// It is a port so that the digest scheme is an infrastructure decision, not a
// use-case one: the use cases only ever hold a code and a digest.
type LoginCodeService interface {
	// Generate returns a fresh code together with the digest to persist. The
	// plaintext is returned once, for the outgoing e-mail, and is never stored.
	Generate() (code string, digest string, err error)

	// Matches reports whether the code produces the stored digest. It must run
	// in time independent of how many leading digits are correct, or the
	// comparison itself leaks the code one digit at a time.
	Matches(digest, code string) bool
}

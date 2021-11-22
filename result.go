package nogo

type Result struct {
	Rule

	// Found is true if any matching rule was found.
	// Do not use it to check if the file is actually to be ignored!
	// For this use Resolve as it takes into account some special cases.
	Found bool

	// ParentMatch saves if the actual rule matched for a parent or not.
	// In case of a parent match the check for OnlyFolder has to be different.
	ParentMatch bool
}

// Resolve the Result by taking into account OnlyFolder
// and if the matched path is a directory.
func (r Result) Resolve(isDir bool) bool {
	if r.Found && r.OnlyFolder && !isDir && !r.ParentMatch {
		return false
	}

	if r.Found && r.Negate {
		return false
	}

	return r.Found
}

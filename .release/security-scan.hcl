# Copyright IBM Corp. 2014, 2026
# SPDX-License-Identifier: MPL-2.0

container {
	dependencies = true
	alpine_secdb = true
	secrets      = true
}

binary {
	secrets      = true
	go_modules   = true
	osv          = true
	oss_index    = false
	nvd          = false
	triage {
		suppress {
			vulnerabilites = [
				"GO-2026-5932",	# golang.org/x/crypto/openpgp is unmaintained but not imported by envconsul
			]
		}
	}
}
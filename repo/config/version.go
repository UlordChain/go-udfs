package config

// CurrentCommit is the current git commit, this is set as a ldflag in the Makefile
var CurrentCommit string = "feat/backup-v0.0.1"

// CurrentVersionNumber is the current application's version literal
const CurrentVersionNumber = "0.4.17"

const ApiVersion = "/go-ipfs/" + CurrentVersionNumber + "/"

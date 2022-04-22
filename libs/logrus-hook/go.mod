module github.com/lancer-kit/uwe/libs/logrus-hook

go 1.17

require (
	github.com/sirupsen/logrus v1.8.1
	github.com/lancer-kit/uwe/v3 v3.0.0
)

require golang.org/x/sys v0.0.0-20191026070338-33540a1f6037 // indirect

replace (
	github.com/lancer-kit/uwe/v3 => ../../
)

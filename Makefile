
tidy:
	go mod tidy
	cd examples/recover && go mod tidy
	cd examples/simpleapi && go mod tidy
	cd examples/simpleapp && go mod tidy
	cd examples/simplecron && go mod tidy
	cd libs/clicheck && go mod tidy
	cd libs/cronjob && go mod tidy
	cd libs/logrus-hook && go mod tidy
	cd libs/zerolog-hook && go mod tidy
module simpleapi

go 1.17

require (
	github.com/go-chi/chi v4.0.2+incompatible
	github.com/lancer-kit/uwe/v3 v3.0.0
)

require github.com/sheb-gregor/sam v1.0.0 // indirect

replace github.com/lancer-kit/uwe/v3 => ../../

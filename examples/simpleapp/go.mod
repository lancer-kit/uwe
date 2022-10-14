module simpleapp

go 1.18

require (
	github.com/go-chi/chi v1.5.4
	github.com/lancer-kit/uwe/v3 v3.0.0
)

require github.com/sheb-gregor/sam v1.0.0 // indirect

replace github.com/lancer-kit/uwe/v3 => ../../

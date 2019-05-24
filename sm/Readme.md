# sm
--
    import "github.com/lancer-kit/uwe/sm"


## Usage

#### type Hook

```go
type Hook func(from, to State) (bool, error)
```


#### type HookList

```go
type HookList struct {
}
```


#### func (HookList) New

```go
func (HookList) New() HookList
```

#### type State

```go
type State string
```


#### type StateMachine

```go
type StateMachine struct {
}
```


#### func  NewStateMachine

```go
func NewStateMachine() StateMachine
```

#### func (*StateMachine) AddTransition

```go
func (sm *StateMachine) AddTransition(from, to State) error
```

#### func (*StateMachine) AddTransitions

```go
func (sm *StateMachine) AddTransitions(from State, to ...State) error
```

#### func (StateMachine) Clone

```go
func (sm StateMachine) Clone() StateMachine
```

#### func (*StateMachine) DoTransition

```go
func (sm *StateMachine) DoTransition(name State) error
```

#### func (*StateMachine) GoBack

```go
func (sm *StateMachine) GoBack() error
```

#### func (StateMachine) New

```go
func (StateMachine) New() StateMachine
```

#### func (*StateMachine) SetState

```go
func (sm *StateMachine) SetState(state State)
```

#### func (*StateMachine) State

```go
func (sm *StateMachine) State() State
```

package main

import (
	"fmt"
)

type MotorBike struct {
	name    string
	engine  int
	gearBox int
	wheels  string
	color   string
	price   float32
}

func (m *MotorBike) getEngine() string {
	return fmt.Sprintf("%v %s", m.engine, "cc")
}

func (m MotorBike) String() string {
	return fmt.Sprintf("MotorBike %s %vcc Engine, %vGears, %s %s at on-road price of %fRs", m.name, m.engine, m.gearBox, m.wheels, m.color, m.price)
}

func main() {
	dominor := MotorBike{name: "Dominor", engine: 400, gearBox: 6, wheels: "Daimon cut alloy", color: "Parrot Green", price: 298733.72}
	fmt.Println(dominor.getEngine())
	fmt.Println(dominor)

}

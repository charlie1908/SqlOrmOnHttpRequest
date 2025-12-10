package RabbitMQ

import (
	"encoding/json"
	"httpRequestName/Core"
	"httpRequestName/Model"
	"log"
)

func ListenUserRabbitMQ() error {
	client, err := Core.NewRabbitMQClient()
	if err != nil {
		panic(err)
	}
	defer client.Close()
	err = client.Consume("newUser", func(user string) {
		var person Model.Person
		err := json.Unmarshal([]byte(user), &person)
		if err != nil {
			log.Printf("Unmarshal error: %v", err)
		}
		log.Printf("User Name: %s | User Age: %d", person.Name, person.Age)
		log.Println("Received a message: ", user)
	})
	if err != nil {
		log.Printf("Consume error: %v", err)
	}
	select {} // Prevent app from exiting
}

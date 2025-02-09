# GO Microservices Observability Example

This repository contains an example of how to instrument a Go microservice with OpenTelemetry and Jaeger. There are four
services in this example:

1. **User Service**: A service that do the permission check of an user.
2. **Order Service**: A service that store orders for an user.
3. **Inventory Service**: A service that check the availability of a product and deduct items for an order.
4. **Notification Service**: A service that send a notification to an user.

The services are connected in the following way:

- The **Order Service** depends on the **User Service** to check if the user has permission to create an order.
- The **Order Service** takes in orders and publish a message to a message queue (in memory) where the **Inventory
  Service** and **Notification Service** is listening. 

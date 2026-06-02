# **ISOMAP (NUS Orbital 2026)**
~ Ken Tay Yao Hui, Srivathsan Ram

--- 

## Table of Contents

- [Project Overview](#Project-Overview)
- [Aim](#Aim)
- [Motivation](#Motivation)
- [User Stories](#User-Stories)
  
- [Proposed Core Features](#Proposed-Core-Features)

- [Technical Proof of Concept](#Technical-Proof-of-Concept)

- [Proposed LOA](#Proposed-Level-of-Achievement)

- [Tech Stack](#Tech-Stack)
- [SWE Practices](#Software-Engineering-Practices)

- [Timeline and Development Plan](#Timeline-and-Development-Plan)

---
## Project Overview
An application developed to bring more to travel to make it more enjoyable and fruitful by integrating exploration and discovery not only for a sole user.

---
## Aim
We aim to help users get to their destination and make the most of their journey.

--- 
## Motivation
Modern navigation tools such as Google Maps Platform are excellent at computing the fastest route from Point A to Point B.

But daily commuting is more than a race to the destination.

---
## User Stories

As a commuter


As a Group of friends


As a tourist

--- 
## Proposed Core Features
- Smart Navigation and Basic Routing
- Travel-Time Proximity 
- Group Planning
- Exploration Feature

---
## Technical Proof-of-Concept

### Basic Routing
To answer the question of "What is the best route to get from point A to B?"

### Travel-Time Proximity 
To answer the question of "What can I reach within 20 mins by bus?" 

To prove that we can obtain a specific range that a person can reach, we first tried to obtain a range of locations that can be reached from a point

This helps us believe that a specific range can then be obtained by filtering the possible routes out slowly one by one 

---
### Proposed Level of Achievement
Level of Achievement: Apollo

---
## Tech Stack
- React (front-end)
- Go (using GORM and Gin)
- Postgres (database)

--- 
## Software Engineering Practices

### Workflow & Tools
- Version Control by utilising repositories and branching in github

### Code Quality * Maintenance
- Maintaining clear separation of concerns between data access, middleware, controller and route layers
- Creating reusable components in React

### Development Strategy 
- Commenting Code and keeping files well documented in the README

### Testing & Reliability
- Unit Testing, Automated Testing

--- 
## Timeline and Development Plan

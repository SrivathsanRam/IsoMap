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

|                                                                 S/N                                                                 	|                                             Tasks                                            	|                                Description                                	|    In Charge   	|        Date       	|
|:-----------------------------------------------------------------------------------------------------------------------------------:	|:--------------------------------------------------------------------------------------------:	|:-------------------------------------------------------------------------:	|:--------------:	|:-----------------:	|
|                                                                  1                                                                  	|                                    Finalise IsoMap Details                                   	| Refine our motivation, goals, and key features                            	| Ken Srivathsan 	|  11 May - 17 May  	|
| Lift-Off: Poster & Video Submission An A4-size poster (<Project ID>.jpg or .png)  A ~1-minute presentation video (<Project ID>.mp4) 	|                                                                                              	|                                                                           	|                	|       18 May      	|
|                                                                   2                                                                 	|                                  Familiarise and Set up Git                                  	| Create a repository and start a README and initial code                   	| Ken Srivathsan 	|  18 May - 24 May  	|
|                                                                  3                                                                  	|                 Familiarise yourself with the relevant TechStack we are using                	| Learn React Basics to settle the front-end                                	|       Ken      	|  25 May - 31 May  	|
|                                                                                                                                     	|                                                                                              	| Learn to set up the back-end                                              	|   Srivathsan   	|                   	|
|      Milestone 1: Ideation Ideation: Features, System Design, Development Plan Technical Proof of Concept* Document the System      	|                                                                                              	|                                                                           	|                	|       1 June      	|
|                                                                  4                                                                  	|                                 Focus on core infrastructure                                 	|      Develop API skeletons and Go project setup with database design      	|   Srivathsan   	|  1 June - 7 June  	|
|                                                                                                                                     	|                                                                                              	|        Set up basic map and navigation and home display with react        	|       Ken      	|                   	|
|                                                                  5                                                                  	|                                  Implementing basic routing                                  	|            Routing API integration and endpoint implementation            	|   Srivathsan   	|  8 June - 14 June 	|
|                                                                                                                                     	|                                                                                              	|                Create search interface & routing display UI               	|       Ken      	|                   	|
|                                                                  6                                                                  	|                                Implementing Proximity Feature                                	| Reachability calculations and transport filtering                         	|       Sri      	| 15 June - 21 June 	|
|                                                                                                                                     	|                                                                                              	| Time selection UI, category filtering, reachability display               	|       Ken      	|                   	|
|                                                                  7                                                                  	|                                Preparation for MS2 submission                                	| Integration testing, bug fixing and doing up the necessary deliverables   	|       Sri      	| 22 June - 28 June 	|
|                                                                                                                                     	|                                                                                              	|                                                                           	|       Ken      	|                   	|
|               Milestone 2: Prototyping  Implement a prototype of the system with core features Perform system testing               	|                                                                                              	|                                                                           	|                	|      29 June      	|
|                                                                  8                                                                  	|                               Implementing Exploration Feature                               	| Exploration endpoints and route-based recommendations                     	|       Sri      	|  29 June - 5 July 	|
|                                                                                                                                     	|                                                                                              	| Exploration panel, route recommendations and deal/scenic display          	|       Ken      	|                   	|
|                                                                  9                                                                  	|                              Implementing Group Planning Feature                             	| Settle meeting point algorithm and Group APIs                             	|       Sri      	|  6 July - 12 July 	|
|                                                                                                                                     	|                                                                                              	| Group Creation UI, Member management and centralised location display     	|       Ken      	|                   	|
|                                                                  10                                                                 	|                                User accounts and saved places                                	| Authentication, saved locations and user management                       	|       Sri      	| 13 July - 19 July 	|
|                                                                                                                                     	|                                                                                              	| Login/Registration, saved places UI and user profile                      	|       Ken      	|                   	|
|                                                                  11                                                                 	|                                        MS3 Submission                                        	| Focus on system testing, user testing and preparing deliverables          	|       Sri      	| 20 July - 26 July 	|
|                                                                                                                                     	|                                                                                              	|                                                                           	|       Ken      	|                   	|
|                           Milestone 3: Extension Extend system with additional features More user testing                           	|                                                                                              	|                                                                           	|                	|      27 July      	|
|                                                                  12                                                                 	| Focus on advanced features: Traffic consideration Public transport crowding Pit-stop routing 	| Research and implement the selected feature of choice                     	|       Sri      	|  27 July - 2 Aug  	|
|                                                                                                                                     	|                                                                                              	|                                                                           	|       Ken      	|                   	|
|                                                                  13                                                                 	|                        Focus on implementing one more advanced feature                       	| Research and implement the selected feature of choice                     	|       Sri      	|   3 Aug - 9 Aug   	|
|                                                                                                                                     	|                                                                                              	|                                                                           	|       Ken      	|                   	|
|                                                                  14                                                                 	|                                Polishing up current features                                 	| Optimising performance and error handling                                 	|       Sri      	|  10 Aug - 16 Aug  	|
|                                                                                                                                     	|                                                                                              	| UI improvements and optimising UI for convenience                         	|       Ken      	|                   	|
|                                                                  15                                                                 	|                                      Release Preparation                                     	| Final testing, bug fixing and demo preparation and deployment preparation 	|       Sri      	|  17 Aug - 25 Aug  	|
|                                                                                                                                     	|                                                                                              	|                                                                           	|       Ken      	|                   	|
|             Splashdown - Refinement Polish up your system and fix outstanding issues Make your final submission required            	|                                                                                              	|                                                                           	|                	|       26 Aug      	|
|                                                       Optional Implementations                                                      	|                                                                                              	|                                                                           	|                	|                   	|
|                                                                  01                                                                 	|                                   Live view implementation                                   	|                                                                           	|                	|                   	|

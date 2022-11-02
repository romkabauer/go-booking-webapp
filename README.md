# Conference managing and booking service

Learning Go by creating this service

## TODO List:
### Persist booking data in the file locally            DONE
### Make web-service                                    15/16
###### POST /login/admin                                DONE
###### POST /login/user
###### GET /conference                                  DONE
###### GET /conference/{id}                             DONE
###### POST /conference                                 DONE
###### DELETE /conference/{id}                          DONE
###### PUT /conference/{id}                             DONE
###### PATCH /conference/{id}/name                      DONE
###### PATCH /conference/{id}/totaltickets              DONE
###### GET /conference/{confId}/booking                 DONE
###### GET /conference/{confId}/booking/{id}            DONE
###### POST /conference/{confId}/booking                DONE
###### PATCH /conference/{confId}/boooking/{id}/cancel  DONE
###### PUT /conference/{confId}/booking/{id}            DONE
###### PATCH /conference/{confId}/booking/{id}/name     DONE
###### PATCH /conference/{confId}/booking/{id}/tickets  DONE
### Cover with tests
### Pack to the Docker container                        DONE
### Store data in local no-SQL DB docker instance
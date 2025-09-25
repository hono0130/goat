sequenceDiagram
    participant ClientStateMachine
    participant ServerStateMachine
    participant DBStateMachine

    ClientStateMachine->>ServerStateMachine: ReservationRequestEvent
    ServerStateMachine->>DBStateMachine: DBSelectEvent
    DBStateMachine->>ServerStateMachine: DBSelectResultEvent
    alt
        ServerStateMachine->>ClientStateMachine: ReservationResultEvent
    else
        ServerStateMachine->>DBStateMachine: DBUpdateEvent
        DBStateMachine->>ServerStateMachine: DBUpdateResultEvent
        ServerStateMachine->>ClientStateMachine: ReservationResultEvent
    end

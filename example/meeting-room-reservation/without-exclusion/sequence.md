sequenceDiagram
    participant ClientStateMachine
    participant ServerStateMachine
    participant DBStateMachine

    ClientStateMachine->>ServerStateMachine: ReservationRequestEvent
    ServerStateMachine->>DBStateMachine: DBSelectEvent
    DBStateMachine->>ServerStateMachine: DBSelectResultEvent
    opt
        ServerStateMachine->>ClientStateMachine: ReservationResultEvent
    end
    opt
        ServerStateMachine->>DBStateMachine: DBUpdateEvent
        DBStateMachine->>ServerStateMachine: DBUpdateResultEvent
        ServerStateMachine->>ClientStateMachine: ReservationResultEvent
    end

sequenceDiagram
    participant ServerStateMachine
    participant DBStateMachine
    participant ClientStateMachine
    ClientStateMachine->>ServerStateMachine: ReservationRequestEvent
    ServerStateMachine->>DBStateMachine: DBSelectEvent
    DBStateMachine->>ServerStateMachine: DBSelectResultEvent
    DBStateMachine->>ServerStateMachine: DBUpdateResultEvent
    ServerStateMachine->>DBStateMachine: DBUpdateEvent
    ServerStateMachine->>ClientStateMachine: ReservationResultEvent
    ServerStateMachine->>ClientStateMachine: ReservationResultEvent

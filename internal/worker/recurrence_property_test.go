package worker

// Feature: subscription-status-log-backend, Property 4: Worker sempre usa EventSource "system"
//
// This property is validated at the service layer in
// internal/service/subscription_property_test.go (TestProperty_SubTransition_WorkerEventSourceIsSystem)
// because RecurrenceService lives in the service package and calls subSvc.Transition
// with TransitionEvent{Source: "system", Reason: <descriptive>}.
//
// The RecurrenceWorker in this package is a queue consumer that delegates all
// billing logic to RecurrenceService — it does not call Transition directly.
// Therefore the property is best expressed and verified at the service layer.

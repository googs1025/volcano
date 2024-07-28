# Volcano Scheduler Session Context Proposal

## Introduction
In the current Volcano scheduler, each time a scheduling cycle is executed, a Session is created, the OpenSession method is called, and the CloseSession method is invoked at the end. 
Actions are carried out sequentially, and plugins are called to implement scheduling algorithms. However, there is currently no unified Context in the Session, leading to widespread use of `context.TODO()` and `context.Background()` throughout the code.

For example, in the Kubernetes framework, the concept of scheduling context is also introduced, utilizing a separate context throughout the scheduling flow.

- [docs](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/#scheduling-cycle-binding-cycle)
- [code](https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/schedule_one.go#L107)

### Advantages of Context:

![ctx](./images/Volcano-Scheduler-Session-Context-2.png?raw=true)


- Cancellation propagation
- Deadline and timeout management
- Request-scoped value passing
- Goroutine concurrency control
- Context chaining
- Standardized API
## Solution
When calling the OpenSession method, we can initialize a new context and pass it throughout the entire scheduling cycle. This approach ensures consistent management of cancellation signals, deadlines, and other context-related values throughout the process, thereby enhancing the reliability and maintainability of the code.

```go
// Session information for the current session
type Session struct {
   UID types.UID


   kubeClient      kubernetes.Interface
   recorder        record.EventRecorder
   cache           cache.Cache
   restConfig      *rest.Config
   informerFactory informers.SharedInformerFactory
   
   sessionContext  context.Context 
   ...

}

// openSession 
func openSession(cache cache.Cache) *Session {
   ssn := &Session{
      UID:             uuid.NewUUID(),
      kubeClient:      cache.Client(),
      restConfig:      cache.ClientConfig(),
      recorder:        cache.EventRecorder(),
      cache:           cache,
      informerFactory: cache.SharedInformerFactory(),
      sessionContext:  context.Background(),

      ...
}

// closeSession closes the session, and cleans up all the resources.
func closeSession(ssn *Session) {
   ...
  
   ssn.sessionContext = nil
   ...


   klog.V(3).Infof("Close Session %v", ssn.UID)
}

```

![sessionContext](./images/Volcano-Scheduler-Session-Context-1.png?raw=true)


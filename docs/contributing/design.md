# Design

[[toc]]

## Portability

Making Kuberik run containers ensures that it is portable. Although current work is mostly targeted on running the workloads on Kubernetes, for other purposes it would also make sense to run it with other engines. For example, if you'd want to test your piplines, it would make sense to enable executing it on your local machine. In this case, implementing plain Docker scheduler (which comes without advanced feature of Kubernetes) would be a good idea.

## Domain specific

While in its core, Kuberik tries to be domain agnostic, i.e. doesn't differ between CD or data processing pipelines, there's a way to extend Kuberik to make it easier to developers to develop their pipelines.

## Testability

Many pipeline engines are plainly said untestable. This comes from the fact that pipelines are generally used as high-level workloads, meaning that they integrate very different software into a one continous execution. Checking if the whole piplines runs in a single go is usually unpractical as they last too long or touch production services.

Kuberik aims to solve the problems of testability by having frame-level testing. This would allow for verifying each of the frames functionality and input/output integrity.

Simplest way it could work is to define test suits. Each test suit runs (in order) an initialization frame, frame under test (FUT) and verification frame, which verifies that FUT completed correctly. It would also allow to run additional mock services if FUT needs access to some external services and can't interact with real ones during test.

## Full Circle

E.g. Kuberik pipeline should be able to test Kuberik locally, run the CI system for itself, and deploy Kuberik itself to production.

## Resiliancy

Many pipeline engines fail the pipeline on transient error, or end up in a dirty state and are unable to recover without manual intervention. When the pipeline gets scheduled it should get executed.

## Screeners
Every pipeline depends on some sort of a trigger. That's what defines _when_ the pipeline should run. This can be a simple UI click, or a webhook.

Goal of Kuberik is to define an API with which standalone screeners can register. Kuberik would be responsible for handling the events and deduplication, while the screeners would simply send the events to Kuberik. A possible implementation of this would be to define a registration endpoint with which all of the screeners can register. This would establish a direct connection with Kuberik and each of the screeners. Kuberik would also define a Trigger CRD, with which concrete triggers are defined. Kuberik would send this information to appropriate Screeners, and send an event for a matching trigger.

## Pipeline defintion

Although Kuberik tries to avoid creating yet another pipeline with a DSL, this is largely unavoidable. However, it is guided by Kubernetes principles, which makes it more of an API than a DSL.

Kuberik workflows are defined as a single YAML file. This file describes the model of the pipeline. It is design for composability and the ability to generate the pipeline on fly.

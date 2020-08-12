<p align="center"><img src="octane-logo.png" alt="Octane Logo"></p>


# Octane: Kubernetes Cost Engine

Octane help Development Teams easily  **manage their cloud spend on Kubernetes**Octane provideds detailed cost attribution of Infrastructure consumption (e.g. cpu, mem, storage) to Kubernetes resources (clusters, namespaces, pods). 

Octane makes it easier to get a real time view into financial spend on your cloud infrastructure. It works on any main cloud provider (AWS, GCP, Azure).

## Core Features

  - Detailed cost attribution for pods per application
  - consolidation of spend across multiple clusters (e.g. aws + gcp cluster cost in a single pane)
  - Filter spend by pod, namespace, cluster
  - Filter spend by Compute and Storage
  - Get % changes day over day of spend changes
  - Cost attribution by Teams (e.g. Security Team spent $400 today)
  ** Coming Soon 
  - GPU Attribution per pod
  - Data Transfer Costs 

## Installation

Reach out to support@getoctane.io to get an octaneKey to begin using the cost engine.

```bash
sed -i 's/octanekey/linux/' kube-config.yaml
```

## Usage

1) Head over to https://www.cloud.getoctane.io

2) Enter the username and password given to you by the Octane Support Team

3) Voila! You should see real-time cost data coming in.

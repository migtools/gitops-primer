[![Validate Primer](https://github.com/cooktheryan/gitops-primer/actions/workflows/validate-primer.yaml/badge.svg)](https://github.com/cooktheryan/gitops-primer/actions/workflows/validate-primer.yaml)

# gitops-primer
GitOps Primer is an operator can be deployed with a Kubernetes environment to extract objects out of the cluster and store them within a Git repository.

## Developing
If you would like to test or develop using GitOps Primer deploy Minikube(https://minikube.sigs.k8s.io/docs/start/) or Kind(https://kind.sigs.k8s.io/) and run the following.

```
make install
make run
```

## Deploying
If you would like to run GitOps primer within your environment. 
```
make deploy
```

## Running
A secret containing an SSH key that is linked to the Git Repository must be created before running GitOps Primer. Follow the steps to add a new SSH key to your GitHub account(https://docs.github.com/en/github/authenticating-to-github/connecting-to-github-with-ssh/adding-a-new-ssh-key-to-your-github-account).

```
oc create secret generic secret-key --from-file=id_rsa=~/.ssh/id_rsa
```

Now that the SSH key is loaded modify the file examples/extract.yaml to define the git branch and repository to use and then deploy.

```
oc create -f examples/extract.yaml
```

After the job completes, items will exist within your git repository.

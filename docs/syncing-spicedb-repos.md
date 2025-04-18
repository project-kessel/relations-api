# Syncing SpiceDB Repositories

We manage two forks from authzed: one for the [SpiceDB](https://github.com/project-kessel/spicedb) project itself and one for the [SpiceDB Operator](https://github.com/project-kessel/spicedb-operator). This document outlines the steps needed to sync our forks with their upstream counterparts.

## Steps to Sync Forked Repositories

Follow these steps to sync your forked repository with the upstream repository:

1. **Add the Upstream Remote**  
    Ensure the upstream repository is added as a remote. Run the following command to add it:  
    ```bash
    git remote add upstream <upstream-repo-url>
    ```  

2. **Fetch Upstream Changes**  
    Fetch the latest changes from the upstream repository:  
    ```bash
    git fetch upstream
    ```

3. **Create a New Branch from the Desired Release Tag**  
    Identify the release tag you want to sync from the upstream repository. Create a new branch based on this tag:  
    ```bash
    git checkout -b sync-upstream-<tag> tags/$TAG
    ```  
    Replace `<tag>` with the desired release tag.

4. **Switch to a New Branch for Rebasing**  
    Create and switch to a new branch from our main where you will apply the rebase:  
    ```bash
    git checkout -b rebase-upstream-<tag> main
    ```

5. **Rebase onto the Sync Branch**  
    Rebase your new branch onto the branch created from the upstream release tag:  
    ```bash
    git rebase sync-upstream-<tag>
    ```

6. **Resolve Any Conflicts**  
    If there are conflicts during the rebase, resolve them manually. After resolving conflicts, continue the rebase process:  
    ```bash
    git rebase --continue
    ```

    You may run into issues with deleted files that we have removed in our fork. If you encounter this, you can skip the commit with the following command:  
    ```bash
    git rebase --skip
    ```

7. **Push the Rebased Branch to Your Fork**  
    Push the rebased branch to your forked repository:  
    ```bash
    git push origin rebase-upstream-<tag>
    ```

8. **Create a Merge Request**  
    Open a merge request in your repository to merge the rebased branch into your main branch. **Ensure you do not squash commits during the merge to preserve the commit history.**

## Post-Sync tips

- **Disable new workflows**: If the upstream repository has added new workflows, we may want to disable them if they are not pertinent to our use cases. You can do this by navigating to the `.github/workflows` directory and adding an `if` condition with a value of `false` to the workflow you want to disable. For example:
    ```yaml
    crdbdatastoreinttest:
        name: "Datastore Integration Tests"
        runs-on: "buildjet-4vcpu-ubuntu-2204"
        needs: "paths-filter"
        if: false
        ...
    ```
    This will show the workflow as "skipped" in the GitHub UI.

- **Ensure enabled workflows are running on `ubuntu-latest`**: If the upstream repository has added new workflows we intend to keep enabled, ensure that they are running on `ubuntu-latest` instead of `buildjet-*`.
    ```yaml
    runs-on: "ubuntu-latest"
    ```

- **Validate CI**: With the PR open, ensure that all workflows pass and verify that everything looks as expected.

- **Ensure Migrations Are Run**: New versions of SpiceDB may include database migrations. You may need to force SpiceDB Operator to run new migrations by pinning the `SPICEDB_IMAGE` parameter in the Relations app-interface SaaS template to the specific commit instead of `latest`. **It is important to note that this may cause downtime** based on how long the migrations take and what they consist of; take note when you roll-out to stage.

***************************************************************
Deploy AWS ECS
***************************************************************

.. contents:: Content
   :depth: 2


Overview
========

 Command line tool to deal with AWS deployment


Install
=======

You can directly use the binaries we provide. For OS X::

    $ wget -O https://github.com/guilherme-santos/deploy-ecs/raw/master/binaries/osx/deploy-ecs
    # sudo mv deploy-ecs /usr/local/bin/

Or for Unix::

    $ wget -O https://github.com/guilherme-santos/deploy-ecs/raw/master/binaries/unix/deploy-ecs
    # sudo mv deploy-ecs /usr/local/bin/


Usage
=====

After that, the following command will be available in your PATH::

    $ deploy-ecs --help

We have the following commands:

* **config**

* **deploy**

* **env**

* **exec**

* **kill**

* **list-revisions**

* **logs**

* **ps**

* **rollback**

* **scale**

* **self-update**

* **task-definition**

If you're inside of a git repository, `deploy-ecs` will get the service name from it, otherwise
you need to use the `-s|--service <service-name>` flag. The service name can be the project name
(e.g. *dbmapping*) or/and use `--repository <url>` (e.g. git@github.com:guilherme-santos/dbmapping.git).


Deploy
------

You can deploy either a tag or a branch from git. This command will build a Docker image from this tag/branch,
update the task definition of this service and restart the service to use this new version.

    $ deploy-ecs deploy -s my-service --tag v1.2.3

You can force deploy a specific revision of this service, for example:

    $ deploy-ecs deploy -s my-service --revision 13

You can also add some optional flags, for example:

* **--rebuild**: will ignore all docker cached layers you have and it'll build the image from scratch

* **--wait**: will wait until service is running and health


Rollback
--------

If for any reason you need to rollback the last deployed version, you can use this command.
It's a shortcut to `deploy-ecs deploy -s my-service --revision <N-1>`:

    $ deploy-ecs rollback -s my-service

Env
-------

You can get the all env vars configured to this service, to format as json use **-json**.

    $ deploy-ecs env -s my-service --json

To update an env var, use **-set**, and to remove use **-unset**. A new task definition will be created:

    $ deploy-ecs env -s my-service --set 'DATABASE_URL=localhost:3306' --set 'DATABASE_NAME=my_service'

To get env vars from a specific revision of a task definition, use the **--revision** flag.

You can use **--deploy** and **--wait** to deploy and wait service be health


Task definition
---------------

You can get all attributes from the last task definition:

    $ deploy-ecs task-definition -s my-service

To update an attribute without deploing this new version you can use **-set**:

    $ deploy-ecs task-definition -s my-service --set 'entryPoint=["sh", "-c"]' --set 'command="echo hello world"'

To get the task definition from a specific revision, use the **--revision** flag.

You can use **--deploy** and **--wait** to deploy and wait service be health


List revisions
--------------

You can list all revisions from a specific service:

    $ deploy-ecs list-revisions -s my-service

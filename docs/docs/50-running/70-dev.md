---
title: Dev Mode 
---
To facilitate interactive development, Acorn provides a `dev` command that enables you to run your project in a live-reload mode, which automatically reloads your application whenever you make changes to your code. To start the interactive mode, you can use the `acorn dev` command.
The interactive dev mode offers a convenient way to test and debug your project as you develop it, as you can quickly see the effects of any changes you make. This mode can also help you to identify and fix errors more efficiently by providing real-time feedback on your code.

- Start a dev session from the current working directory
  - ```bash
    acorn dev
    ```
- Start a dev session from a pre-built image in the current working directory
  - ```bash
    acorn dev [IMAGE]
    ```
- Attach a dev session to a pre-existing acorn in the current working directory
  - ```bash
    acorn dev -n [APP_NAME]
    ```

:::note
When using `acorn dev`, the `dev` profile will be automatically used. This behavior is the same when using `acorn run -i`.
:::
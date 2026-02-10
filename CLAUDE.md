- Golang code files should typically be no longer then 300 lines, max 400
- When you suggest code, also update or add the tests
- When you change code also update the documentation
- This tool is meant for developers - making it easy to use
- In the README I want to have the quick path first and the more advanced further
- When I change cli commands , make sure the completion and the docs are up to date
- When a config setting has multiple values , consider splitting them into different code files
- Make sure the help allow people to easy can available options

- podman tmpfs has no userid and guid params
- docker and orbstack have userid and guid params
- podman can't access sockets 
 On macOS, podman runs containers inside a Linux VM. Unix sockets can't be shared
   from macOS into the VM via virtiofs, so mounting the SSH proxy socket fails with statfs 
  ... operation not supported.  
- docker and orbstack DO support sockets

- for addt tests use the cli directly avoid using the code functions directly
- for addt tests do things in a temp directory and use the config commands to activate the right features for the test     
- for addt test use the command root to invoke the cli , instead of shelling , this allows us to test the current version   
- for addt tests use temporary directories of your creating files for testing
- for addt tests , we'll be testing both providers (when available)
- for addt tests you can look at debug logs by setting the loglevel
- for addt tests avoid running any provider commands directly
- for addt tests when you want to run a shell command change the addt_command, for extensions specific tests keep the existing command
- for addt tests , make them read as user scenarios in the comments
- for addt tests you typically use a testdata debug extension not one of the extensions itself
- NEVER use a timeout below 600s for addt tests — image builds take 1-2 minutes each
- before running addt tests - best to cleanup the stale running containers
- Avoid running docker commands directly use the provider dockerCmd
- When you using CLI commands for managing the containers or images, you need to use the right DOCKER_CONTEXT for the provider
- when you create paths for sockets make sure they are unique and don't overlap
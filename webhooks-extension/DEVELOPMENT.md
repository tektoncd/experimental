# Developing

If modifying the UI code and wanting to deploy your updated version:

1) Run `npm run build` this will create a new file in the dist directory
2) Update config/extension-service.yaml to reference this newly created file in the dist directory, this should simply be a matter of changing the hash value - do not change the web/ prefix

    `tekton-dashboard-bundle-location: "web/extension.214d2396.js"`

3) Reinstall (If not re-installing the main dashboard, you will need to kill the dashboard pod after the new extension pod starts.  This only needs doing until https://github.com/tektoncd/dashboard/issues/215 is completed.)

### Linting

Run `npm run lint` to execute the linter. This will ensure code follows the conventions and standards used by the project.

Run `npm run lint:fix` to automatically fix a number of types of problem including code style.
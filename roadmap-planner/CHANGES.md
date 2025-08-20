# Changes

Lets do a number of changes in this code base, we will start with the bellow UI changes

### UI Changes

- [X] Quarters: Make the user select visible quarters with a checkbox and only three should be visible at once
- [X] Lets make each Epic card much smaller with name, priority, ID and status only two rows with less padding and margin to accomodate more data in the screen. We also need it to have a link that will open in a new tab to the original detailed view of the issue
- [X]  Make the milestone card smaller with the summary only, also need a link to the original detailed view of the issue
- [X]  Change the Milestone to encapsulate epic cards making it clear and move the create epic button after all the existing epics in the milestone
- [X]  Correct this bug: For some reason it only displays one milestone per pillar in a quarter, fix it to show all milestones in the quarter
- [X] The assignable api was generating an error, after debugging and adjusting we discovered the assignable endpoint needs an issue key as parameter. You will change the API document and frontend and add the issue key if changing an exisiting issue, and the parrent issue or related issue key if creating a new issue.
- [X] Change the assigne component to use one that can filter the results. The API also supports adding a filter query that will filter in the API itself
- [X] After filtering or selecting the assignee the UI is still saying the assignee is required. Please fix this
- [X] Creating milestone is not working with this error: `Failed to create milestone: failed to create milestone: request failed. Please analyze the request body for more details. Status code: 400 (Response: {"errorMessages":[],"errors":{"assignee":"expected Object containing a 'name' property"}})`. This probably means we should change the assignee payload to be an object with name and key instead of just the account id.
- [X] Make quarter selection "persistent" in frontend, and do the best to save the state


### Backend changes

- [X] Refactor the backend to use Uber's zap logger and make it the default logger, create one instance that should be configured centrally and used in all parts of the system without having to instantiate other standalone loggers.



## Improvements
- [X] Milestones card should remove margin on the 2nd item forward
- [X] Change the header and Product roadmap to enable adding more functionality in the future (like a pending epics list?), move the logout button to the header instead of the board, make the product roadmap filter/header more compact.
- [X] Dragging and dropping epics is almost unusable because of conflicts with text spaces. Move the draggable part to the left of the card and make easier to drag. The card should only be dragged for there.
- [X] Pillars and milestones have a sequence attribute in jira. Make sure all APIs return this data and the UI uses it for sorting pillar and milestones. For Epic sorting please use the fix version field if available with blanks displaying at the top. Make sure that the Kanban also sorts quarters and they are displayed in order (older on the left, newer on the right)
- [X] Upgrade all UI dependencies to newer versions and make any necessary changes
- [X] Create basic milestone update modal for quickly changing the name and quarter of a milestone
- [X] Create a basic epic move modal to quickly "move" epic between milestones
- [X] Create a basic epic update modal for quickly changing the name, component, version, priority and assignee of an epic
- [X] Create an API to fetch only the most basic data which does not change often like pillars, existing versions, quarters. The quarters field is actually a custom field in jira, and in our instance it is the `customfield_12242`. It is necessary to test and find ways to load its values dynamically from Jira, if not possible we can load it from current milestone data or a static list in the configuration file. The whole idea is to reduce initial load time and make each API call faster and easier to process, making the app more reactive. You can also add or change the optmizations as you find to be more productive or find a better solution to improve user experience
- [X] Create another API to fetch data like milestones of an pillar, epics of a milestone, etc. that will be used more often to refresh specific areas of the board when the user changes the data
- [X] Change the UI to load the basic data first (after the backend is already optimized) and display data
- [X] Make (create milestones) modal load the specific quarter it was clicked on.
- [X] Refactor the API and UI optmizations, current issues:
  - [X] UI is doing multiple calls to load all the data which did not improve startup time. Please make sure only one call to the basic api is done when starting the app, all subsequent calls should be done after the app is loaded.
  - [X] Epics and milestone APIs should support filtering instead of making it a subresource API for pillars and milestones. This way we can load milestones from multiple pillars at once, we can also load epics from multiple milestones at once.
  - [X] After UI calls the UI is not displaying milestones or epics. This should be checked and fixed
- [X] Implement smart partial updates and eliminate redundant requests:
  - [X] Smart refresh system: Only refresh affected pillars/milestones after CRUD operations
  - [X] Request deduplication: Prevent multiple identical API calls
  - [X] Optimistic updates: Update UI immediately for better user experience
  - [X] Batch operations: Efficient handling of multiple related updates

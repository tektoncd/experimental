import {FETCH_TASK_DESCRIPTION} from '../Actions/TaskActionType';
import {FETCH_TASK_YAML} from '../Actions/TaskActionType';

// eslint-disable-next-line require-jsdoc
export function fetchTaskDescription(rawURL: string) {
  const readmeURL = rawURL.substring(0, rawURL.lastIndexOf('/') + 1);

  return (dispatch: any) => {
    fetch(`${readmeURL}/README.md`)
      .then((response) => response.text())
      .then((TaskDescription) => dispatch({
        type: FETCH_TASK_DESCRIPTION,
        payload: TaskDescription,
      }));

    fetch(`${rawURL}`)
      .then((response) => response.text())
      .then((TaskYaml) => dispatch({
        type: FETCH_TASK_YAML,
        payload: TaskYaml,
      }));
  };
}

export default fetchTaskDescription;

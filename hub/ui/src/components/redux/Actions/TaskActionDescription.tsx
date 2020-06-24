import {FETCH_TASK_DESCRIPTION} from '../Actions/TaskActionType';
import {FETCH_TASK_YAML} from '../Actions/TaskActionType';

// eslint-disable-next-line require-jsdoc
export function fetchTaskDescription(rawUrl: string) {
  const readmeUrl = rawUrl.substring(0, rawUrl.lastIndexOf('/') + 1);

  return (dispatch: any) => {
    fetch(`${readmeUrl}/README.md`)
      .then((response) => response.text())
      .then((TaskDescription) => dispatch({
        type: FETCH_TASK_DESCRIPTION,
        payload: TaskDescription,
      }));

    fetch(`${rawUrl}`)
      .then((response) => response.text())
      .then((TaskYaml) => dispatch({
        type: FETCH_TASK_YAML,
        payload: TaskYaml,
      }));
  };
}

export default fetchTaskDescription;

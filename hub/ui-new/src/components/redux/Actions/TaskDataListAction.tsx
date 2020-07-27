import {FETCH_TASK_LIST} from '../Actions/TaskActionType';
import {API_URL} from '../../../constants';

// eslint-disable-next-line require-jsdoc
export function fetchTaskList() {
  return (dispatch: any) => {
    fetch(`${API_URL}/resources`)
      .then((response) => response.json())
      .then((TaskData) =>
        dispatch({
          type: FETCH_TASK_LIST,
          payload: TaskData.data,
        }));
  };
}

export default fetchTaskList;

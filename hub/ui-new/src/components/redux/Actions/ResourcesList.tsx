import {FETCH_RESOURCE_LIST} from '../Actions/TaskActionType';
import {API_URL} from '../../../constants';

// eslint-disable-next-line require-jsdoc
export function fetchResourceList() {
  return (dispatch: any) => {
    fetch(`${API_URL}/resources`)
      .then((response) => response.json())
      .then((TaskData) =>
        dispatch({
          type: FETCH_RESOURCE_LIST,
          payload: TaskData.data,
        }));
  };
}

export default fetchResourceList;
;

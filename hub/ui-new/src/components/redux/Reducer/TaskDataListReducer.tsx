import {FETCH_TASK_LIST} from '../Actions/TaskActionType';
const reducer = (state = [], action: any) => {
  switch (action.type) {
    case FETCH_TASK_LIST:
      return {
        ...state,
        TaskDataList: action.payload,
      };
    default: return state;
  }
};

export default reducer;


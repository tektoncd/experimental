import {FETCH_TASK_DESCRIPTION} from '../Actions/TaskActionType';
import {FETCH_TASK_YAML} from '../Actions/TaskActionType';

export interface ResData {
  Name: string,
  Description: string,
  Rating: number,
  Downloads: number,
  YAML: string
}
// type obj = ResData[]
const initialState = {
  data: [],
};

const reducer = (state = initialState, action: any) => {
  switch (action.type) {
    case FETCH_TASK_DESCRIPTION:
      return {
        ...state,
        TaskDescription: action.payload,
      };
    case FETCH_TASK_YAML:
      return {
        ...state,
        TaskYaml: action.payload,
      };
    default: return state;
  }
};

export default reducer;

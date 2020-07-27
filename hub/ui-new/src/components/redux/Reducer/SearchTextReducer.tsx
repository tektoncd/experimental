import {SEARCH_TEXT} from '../Actions/TaskActionType';

const reducer = (state = [], action: any) => {
  switch (action.type) {
    case SEARCH_TEXT:
      return {
        ...state,
        SearchedText: action.payload,
      };
    default: return state;
  }
};

export default reducer;

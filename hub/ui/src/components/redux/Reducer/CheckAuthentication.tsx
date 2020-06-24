import {CHECK_USER_AUTHENTICATION} from '../Actions/TaskActionType';
let checkAuth: boolean;
if (localStorage.getItem('token') !== null) {
  checkAuth = true;
} else {
  checkAuth = false;
}
const initialState = {
  isAuthenticated: checkAuth,
};

const reducer = (state = initialState, action: any) => {
  switch (action.type) {
    case CHECK_USER_AUTHENTICATION:
      return {
        ...state,
        isAuthenticated: action.payload,
      };
    default: return state;
  }
};

export default reducer;

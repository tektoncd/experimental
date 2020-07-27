/* eslint indent: [2, 4, {"SwitchCase": 1}]*/
// (TODO) update eslint
import {FETCH_RESOURCE_LIST} from '../Actions/TaskActionType';
const reducer = (state = [], action: any) => {
    switch (action.type) {
        case FETCH_RESOURCE_LIST:
            return {
                ...state,
                ResourceList: action.payload,
            };
        default: return state;
    }
};

export default reducer;

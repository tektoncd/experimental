import {combineReducers} from 'redux';
import TaskReducer from './TaskReducer';
import TaskReducerName from './TaskReducerName';
import TaskReducerDescription from './TaskReducerDescription';
import CheckAuthentication from './CheckAuthentication';
import TaskDataListReducer from './TaskDataListReducer';
import ResourceListReducer from './ResourceListReducer';
import SearchTextReducer from './SearchTextReducer';

export default combineReducers({
  TaskData: TaskReducer,
  TaskDataList: TaskDataListReducer,
  ResourceList: ResourceListReducer,
  TaskName: TaskReducerName,
  TaskDescription: TaskReducerDescription,
  TaskYaml: TaskReducerDescription,
  isAuthenticated: CheckAuthentication,
  SearchedText: SearchTextReducer,
});



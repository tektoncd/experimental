import React, {useState} from 'react';
import {DropdownItem, Dropdown, DropdownToggle} from '@patternfly/react-core';
import {connect} from 'react-redux';
import store from '../redux/store';
import fetchTaskSuccess from '../redux/Actions/TaskAction';
import {FETCH_TASK_SUCCESS} from '../redux/Actions/TaskActionType';
export interface TaskPropData {
  id: number;
  name: string,
  description: string,
  rating: number,
  tags: [],
  lastUpdatedAt: string;
  latestVersion;
  catalog: [],
  type: string,
  displayName: string
}
const Sort: React.FC = (props: any) => {
  const [sort, setSort] = useState('Name');
  let tempArr: any = [];
  if (props.TaskData) {
    tempArr = props.TaskData.map((task: any) => {
      const taskData: TaskPropData = {
        id: task.id,
        catalog: task.catalog,
        name: task.name,
        description: task.description,
        rating: task.rating,
        tags: task.tags,
        type: task.type,
        lastUpdatedAt: task.lastUpdatedAt,
        latestVersion: task.latestVersion,
        displayName: task.displayName,
      };
      return taskData;
    });
  }
  function sortByName(event: any) {
    setSort(event.target.text);
    const taskarr = tempArr.sort((first: any, second: any) => {
      if (first.name.toLowerCase() > second.name.toLowerCase()) {
        return 1;
      } else {
        return -1;
      }
    });
    store.dispatch({type: FETCH_TASK_SUCCESS, payload: taskarr});
  }
  // eslint-disable-next-line require-jsdoc
  function sortByRatings(event: any) {
    setSort(event.target.text);
    const taskarr = tempArr.sort((first: any, second: any) => {
      if (first.rating < second.rating) {
        return 1;
      } else {
        return -1;
      }
    });
    store.dispatch({type: FETCH_TASK_SUCCESS, payload: taskarr});
  }


  // Dropdown menu
  const [isOpen, set] = useState(false);
  const dropdownItems = [
    <DropdownItem key="name" onClick={sortByName}>Name</DropdownItem>,
    <DropdownItem key="Rating" onClick={sortByRatings}>Ratings</DropdownItem>,
  ];
  const ontoggle = (isOpen: React.SetStateAction<boolean>) => set(isOpen);
  const onSelect = () => set(!isOpen);


  return (
    <div style={{backgroundColor: 'white'}}>
      <Dropdown
        onSelect={onSelect}
        toggle={<DropdownToggle onToggle={ontoggle}>{sort}</DropdownToggle>}
        isOpen={isOpen}
        dropdownItems={dropdownItems}
      />

    </div>
  );
};
const mapStateToProps = (state) => ({
  TaskData: state.TaskData.TaskData,

});
export default connect(mapStateToProps, {fetchTaskSuccess})(Sort);

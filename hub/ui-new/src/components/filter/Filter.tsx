/* eslint-disable consistent-return */
import React,
{
  useState,
  useEffect,
}
  from 'react';
import {
  Checkbox,
  Button,
} from '@patternfly/react-core/dist/js/components';
import {fetchResourceList} from '../redux/Actions/ResourcesList';
import {API_URL} from '../../constants';
import Sort from './Sort';
import {
  DomainIcon,
  BuildIcon,
  CatIcon,
  CertificateIcon,
  UserIcon,
  TimesIcon,
} from '@patternfly/react-icons';
import store from '../redux/store';
import {
  FETCH_TASK_SUCCESS,
  FETCH_TASK_LIST,
}
  from '../redux/Actions/TaskActionType';
import './filter.css';
import {FlexModifiers, Flex, FlexItem} from '@patternfly/react-core';
import {connect} from 'react-redux';
const tempObj: any = {};
const Filter: React.FC = (props: any) => {
  const [categoriesList, setCategoriesList] = useState();
  const [status, setStatus] = useState({checklist: []});
  const filterItem: any = [
    {
      id: '1000', value: 'task', isChecked: false,
    },
    {
      id: '1001', value: 'pipeline', isChecked: false,
    },
    {
      id: '1002', value: 'Official', isChecked: false,
    },
    {
      id: '1003', value: 'Verified', isChecked: false,
    },
    {
      id: '1004', value: 'Community', isChecked: false,
    }];
  const [checkBoxStatus, setCheckBoxStatus] = useState(
    {},
  );

  // hooks for handling clear button for each filter type
  const [clear, setClear] = useState(true);
  // ////////


  //  function for adding categories to filteritem
  const addCategory = (categoryData: any) => {
    categoryData.map((categoryName: string, index: number) =>
      filterItem.push(
        {
          id: `${categoryName['id']}`,
          value: categoryName['name'], isChecked: false,
        },
      ));
    setStatus({checklist: filterItem});
    return categoryData;
  };
  useEffect(() => {
    // fetchResourceList();
    // fetchTaskList();
    fetch(`${API_URL}/categories`)
      .then((res) => res.json())
      .then((categoryData) =>
        setCategoriesList(addCategory(categoryData)));
    if (categoriesList) {
      (Object.keys(categoriesList)).map((category) => {
        return tempObj.category = false;
      });
    }
    setCheckBoxStatus(tempObj);

    // eslint-disable-next-line
  }, []);


  // / function for showing types
  const addIcon = (idx: number) => {
    switch (idx) {
      case 0:
        return <BuildIcon size="sm" color="black"
          style={{marginLeft: '-0.5em', verticalAlign: '-0.15em'}} />;
      case 1:
        return <DomainIcon size="sm" color="black"
          style={{marginLeft: '-0.5em', verticalAlign: '-0.15em'}} />;
      case 2:
        return <CatIcon size="sm" color="#484848"
          style={{marginLeft: '-0.5em', verticalAlign: '-0.15em'}} />;
      case 3:
        return <CertificateIcon size="sm" color="#484848"
          style={{marginLeft: '-0.5em', verticalAlign: '-0.15em'}} />;
      case 4:
        return <UserIcon size="sm" color="#484848"
          style={{marginLeft: '-0.5em', verticalAlign: '-0.15em'}} />;
      default:
        return;
    }
  };


  // custom label for type filter
  const customLabel = (typeName: string, index: any) => {
    return <Flex>
      <FlexItem breakpointMods={[{modifier: FlexModifiers['spacer-xs']}]}>
        {addIcon(index)}
      </FlexItem>
      <FlexItem>
        {typeName}
      </FlexItem>
    </Flex>;
  };

  // get typed text in search


  // formation of filter url  for calling filterAPi to
  //  fetching task and pipelines
  const filterApi = (event: any) => {
    const tagsList: any = [];
    const searchedtext = Object.values(store.getState().SearchedText);
    const filteredDataList = new Set();
    const resourcetypeList: any = [];
    const resourceVerificationList: any = [];
    const target = event.target;
    // for handling isChecked parameter of checkbox
    setCheckBoxStatus({...checkBoxStatus, [target.value]: target.checked});
    status.checklist.forEach((it: any) => {
      if (it.id === event.target.id) {
        return it.isChecked = event.target.checked;
      }
    },
    );
    status.checklist.slice(0, 2).forEach((item: any) => {
      if (item.isChecked === true) {
        resourcetypeList.push(item.value);
      }
    });
    status.checklist.slice(2, 5).forEach((item: any) => {
      if (item.isChecked === true) {
        resourceVerificationList.push(item.value);
      }
    });
    status.checklist.slice(5).forEach((item: any) => {
      if (item.isChecked === true) {
        categoriesList.forEach((categorytagList: any) => {
          if (categorytagList.name === item.value) {
            categorytagList.tags.forEach((tags: any) =>
              tagsList.push(tags.name));
          }
        });
      }
    });
    tagsList.forEach((tagname: any) => {
      props.ResourceList.forEach((resourceItem: any) => {
        resourceItem.tags.forEach((item: any) => {
          if (item.name === tagname) {
            filteredDataList.add(resourceItem);
            return;
          }
        });
      });
    });
    let filterArray = Array.from(filteredDataList);
    if (tagsList.length === 0) {
      filterArray = props.ResourceList;
    }
    const tempv: any = [];
    if (resourcetypeList.length > 0) {
      resourcetypeList.forEach((resourceType: any) => {
        filterArray.forEach((resourceItem: any) => {
          if (resourceItem.kind.toLowerCase() === resourceType) {
            tempv.push(resourceItem);
          }
        },

        );
      });
      filterArray = tempv;
    }
    const tempx: any = [];
    if (resourceVerificationList.length > 0) {
      resourceVerificationList.forEach((resourceVerification: any) => {
        filterArray.forEach((resourceItem: any) => {
          if (resourceItem.catalog.type ===
            resourceVerification.toLowerCase()) {
            tempx.push(resourceItem);
          }
        });
      });
      filterArray = tempx;
    }
    if (searchedtext[0] !== '') {
      let suggestions: any = [];
      const regex = new RegExp(`${searchedtext[0]}`, 'i');
      suggestions = filterArray.sort().filter((v: any) => regex.test(v.name));
      store.dispatch(
        {
          type: FETCH_TASK_SUCCESS,
          payload: suggestions,
        });
      store.dispatch(
        {
          type: FETCH_TASK_LIST,
          payload: filterArray,
        });
    } else {
      store.dispatch(
        {
          type: FETCH_TASK_SUCCESS,
          payload: filterArray,
        });
      store.dispatch(
        {
          type: FETCH_TASK_LIST,
          payload: filterArray,
        });
    }

    // // for displaying clear filter options
    let flag: any = false;

    status.checklist.forEach((it: any) => {
      if (it.isChecked === true) {
        flag = true;
      }
    });
    if (flag === true) {
      setClear(false);
    } else {
      setClear(true);
    }
  };

  //   function for clearing all checkbox
  const clearFilter = () => {
    const searchedtext = Object.values(store.getState().SearchedText);
    setCheckBoxStatus(
      tempObj,
    );
    status.checklist.forEach((it: any) => {
      it.isChecked = false;
    });

    setClear(true);
    if (searchedtext[0] !== '') {
      let suggestions: any = [];
      const regex = new RegExp(`${searchedtext[0]}`, 'i');
      suggestions = props.ResourceList.sort().filter(
        (v: any) => regex.test(v.name));
      store.dispatch(
        {
          type: FETCH_TASK_SUCCESS,
          payload: suggestions,
        });
      store.dispatch({
        type: FETCH_TASK_LIST,
        payload: props.ResourceList,
      });
    } else {
      store.dispatch(
        {
          type: FETCH_TASK_SUCCESS,
          payload: props.ResourceList,
        });
      store.dispatch(
        {
          type: FETCH_TASK_LIST,
          payload: props.ResourceList,
        });
    }
  };


  // ///////////

  let resourceType: any;
  if (status !== undefined && checkBoxStatus !== undefined) {
    const resource = status.checklist.slice(0, 2);
    resourceType = resource.map((it: any, idx: number) => (
      <div key={`res-${idx}`} style={{marginBottom: '0.5em'}}>
        <Checkbox
          onClick={filterApi}
          isChecked={checkBoxStatus[it.value]}
          style={{width: '1.2em', height: '1.2em', marginRight: '.3em'}}
          label={customLabel(it.value[0].toUpperCase() +
            it.value.slice(1), idx)}
          value={it.value}
          name="type"
          id={it.id}
          aria-label="uncontrolled checkbox example"

        />
      </div>
    ));
  }
  let showverifiedtask: any;
  // jsx element for show verifiedtask
  if (status !== undefined && checkBoxStatus !== undefined) {
    const verifiedtask = status.checklist.slice(2, 5);
    showverifiedtask = verifiedtask.map((it: any, idx: number) => (
      <div key={`task-${idx}`} style={{marginBottom: '0.5em'}}>
        <Checkbox
          onClick={filterApi}
          isChecked={checkBoxStatus[it.value]}
          style={{width: '1.2em', height: '1.2em', marginRight: '.3em'}}
          label={customLabel(it.value[0].toUpperCase() +
            it.value.slice(1), idx + 2)}
          value={it.value}
          name="verification"
          id={it.id}
          aria-label="uncontrolled checkbox example"

        />
      </div>
    ));
  }
  // jsx element for showing all categories
  let categoryList: any = '';
  if (status !== undefined && checkBoxStatus !== undefined) {
    const tempstatus = status.checklist.slice(5);
    tempstatus.sort((a: any, b: any) =>
      (a.value > b.value) ? 1 :
        ((b.value > a.value) ? -1 : 0));
    categoryList =
      tempstatus.map((it: any, idx: number) => (
        <div key={`cat-${idx}`} style={{marginBottom: '0.5em'}}>
          <Checkbox
            onClick={filterApi}
            isChecked={checkBoxStatus[it.value]}
            style={{width: '1.2em', height: '1.2em'}}
            label={it.value[0].toUpperCase() + it.value.slice(1)}
            value={it.value}
            name="tags"
            id={it.id}
            aria-label="uncontrolled checkbox example"

          />
        </div>
      ));
  }

  return (
    <div className="filter-size">
      <Flex style={{marginBottom: '4em'}}>
        <FlexItem >
          <b style={{
            fontSize: '1.1em', verticalAlign: '-0.2em',
            color: '#484848',
          }}>
            Sort
          </b>
        </FlexItem>

        <FlexItem>
          <Sort />
        </FlexItem>
      </Flex>
      <Flex>
        <FlexItem>
          <b style={{fontSize: '1.1em', color: '#484848'}}>Refine By :</b>
        </FlexItem>
        <FlexItem >
          <Button variant='plain'
            isBlock={true}
            isDisabled={clear}
            onClick={clearFilter}>
            <TimesIcon />
          </Button>

        </FlexItem>
      </Flex >
      <Flex>
        <FlexItem style={{marginBottom: '0.3em'}}>
          <b>Kind</b>
        </FlexItem>
      </Flex>
      {resourceType}
      <Flex style={{marginTop: '1.5em'}}>
        <FlexItem style={{marginBottom: '0.3em'}}>
          <b>Support Tier </b>
        </FlexItem>
      </Flex>
      {showverifiedtask}
      <Flex style={{marginTop: '1.5em'}}>
        <FlexItem style={{marginBottom: '0.3em'}}>
          <b>
            Categories
          </b>
        </FlexItem>
      </Flex>
      {categoryList}
    </div >
  );
};

const mapStateToProps = (state: any) => ({
  ResourceList: state.ResourceList.ResourceList,
  TaskDataList: state.TaskDataList.TaskDataList,

});
export default connect(mapStateToProps,
  fetchResourceList)(Filter);

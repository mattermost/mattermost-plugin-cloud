
import React, {useEffect} from 'react';

import PropTypes from 'prop-types';
import {makeStyleFromTheme, changeOpacity} from 'mattermost-redux/utils/theme_utils';
import {WithTooltip} from '@mattermost/shared/components/tooltip';

const SidebarLeft = ({id, installs, actions, showRHS, theme}) => {
    useEffect(() => {
        actions.getCloudUserData(id);
    }, [id, actions]);

    const style = getStyle(theme);

    return (
        <div className='CloudPlugin__SidebarLeft'>
            <WithTooltip
                id='yourCloudInstallationsToolTip'
                title='Your Cloud installations'
                forcedPlacement='right'
            >
                <button
                    type='button'
                    data-testid={'yourCloudInstallationsTestId'}
                    style={style}
                    onClick={() => {
                        showRHS(true);
                    }}
                >
                    <i className='fa fa-cloud'/>
                    {' ' + installs.length}
                </button>
            </WithTooltip>
        </div>
    );
};

SidebarLeft.propTypes = {
    id: PropTypes.string.isRequired,
    installs: PropTypes.array.isRequired,
    actions: PropTypes.shape({
        getCloudUserData: PropTypes.func.isRequired,
    }),
    theme: PropTypes.object.isRequired,
    showRHS: PropTypes.func.isRequired,
};

const getStyle = makeStyleFromTheme((theme) => {
    return {
        color: changeOpacity(theme.sidebarText, 0.6),
        display: 'block',
        padding: 0,
        border: 'none',
        background: 'none',
        cursor: 'pointer',
        textAlign: 'left',
        marginBottom: '10px',
        width: '100%',
    };
});

export default SidebarLeft;
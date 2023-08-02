
import React, {useEffect} from 'react';

import PropTypes from 'prop-types';
import {makeStyleFromTheme, changeOpacity} from 'mattermost-redux/utils/theme_utils';
import {Tooltip, OverlayTrigger} from 'react-bootstrap';

const SidebarLeft = ({id, installs, actions, showRHS, theme}) => {
    useEffect(() => {
        actions.getCloudUserData(id);
    }, [id, actions]);

    const style = getStyle(theme);

    return (
        <div className='CloudPlugin__SidebarLeft'>
            <OverlayTrigger
                placement={'right'}
                overlay={<Tooltip id='yourCloudInstallationsToolTip'>{'Your Cloud installations'}</Tooltip>}
            >
                <a
                    data-testid={'yourCloudInstallationsTestId'}
                    style={style}
                    onClick={() => {
                        showRHS(true);
                    }}
                >
                    <i className='fa fa-cloud'/>
                    {' ' + installs.length}
                </a>
            </OverlayTrigger>
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
        marginBottom: '10px',
        width: '100%',
    };
});

export default SidebarLeft;
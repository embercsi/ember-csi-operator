#!/bin/env python
# Takes input from command: ember-list-drivers -d
# Use CONSOLE_VERSION to define the console version that will be used, because
# available features depend on it
# Use DEVELOPMENT env variable to say we want to use the template-dev.yaml file
import collections
import copy
import itertools
import json
import re
import os
import sys
import yaml


# We don't support NFS drivers right now
EXCLUDE_DRIVERS = ['.*?nfs.*?', '.*vmware.*']
INCLUDE_DRIVERS = None

DEVELOPMENT = bool(int(os.environ.get('DEVELOPMENT', 0) or 0))
ADDITIONAL_SPACES = 2 if DEVELOPMENT else 0

# v1,v2,v3
TRANSFORM_LIST_OF_STRINGS = '__transform_csv'
# k1:v1,k2:v2
TRANSFORM_DICT_OF_STRING = '__transform_csv_kvs'
# Empty string means None
TRANSFORM_POSSIBLE_NONE = '__transform_empty_none'
# Transform a string to a float
TRANSFORM_STRING_FLOAT = '__transform_string_float'


MISSING_OPTIONS = (
    {"default": "$my_ip",
     "deprecated_for_removal": "False",
     "help": "The IP address that the iSCSI daemon is listening on",
     "name": "target_ip_address",
     "required": "False",
     "secret": "False",
     "type": "String"},
    {"default": "admin",
     "deprecated_for_removal": "False",
     "help": "Username for SAN controller",
     "name": "san_login",
     "required": "True",
     "secret": "False",
     "type": "String"},
    {"default": "",
     "deprecated_for_removal": "False",
     "help": "Password for SAN controller",
     "name": "san_password",
     "required": "True",
     "secret": "True",
     "type": "String"},
    {"default": "",
     "deprecated_for_removal": "False",
     "help": "IP address of SAN controller",
     "name": "san_ip",
     "required": "True",
     "secret": "False",
     "type": "String"},
    # target_protocol is defined in MacroSAN FC and ICSCI, but just in case
    {"default": "iscsi",
     "deprecated_for_removal": "False",
     "help": "Determines the target protocol for new volumes, created with "
             "tgtadm, lioadm and nvmet target helpers. In order to enable "
             "RDMA, this parameter should be set with the value \"iser\". The "
             "supported iSCSI protocol values are \"iscsi\" and \"iser\", in "
             "case of nvmet target set to \"nvmet_rdma\".",
     "name": "target_protocol",
     "required": "False",
     "secret": "False",
     "type": "String(choices=['iscsi', 'iser', 'nvmet_rdma'])"},
    # target_protocol is defined in MacroSAN FC and iCSCI, but just in case
    {"default": "tgtadm",
     "deprecated_for_removal": "False",
     "help": "Target user-land tool to use. tgtadm is default, use lioadm for "
             "LIO iSCSI support, scstadmin for SCST target support, ietadm "
             "for iSCSI Enterprise Target, iscsictl for Chelsio iSCSI Target, "
             "nvmet for NVMEoF support, spdk-nvmeof for SPDK NVMe-oF, or fake "
             "for testing.",
     "name": "target_helper",
     "required": "False",
     "secret": "False",
     "type": "String(choices=['tgtadm', 'lioadm', 'scstadmin', 'iscsictl', "
             "'ietadm', 'nvmet', 'spdk-nvmeof', 'fake'])"},
    {"default": "False",
     "deprecated_for_removal": "False",
     "help": "Tell driver to use SSL for connection to backend storage if the "
             "driver supports it.",
     "name": "driver_use_ssl",
     "required": "False",
     "secret": "False",
     "type": "Boolean"},
    {"default": "22",
     "deprecated_for_removal": "False",
     "help": "SSH port to use with SAN",
     "name": "san_ssh_port",
     "required": "False",
     "secret": "False",
     "type": "Port"},
    {"default": "30",
     "deprecated_for_removal": "False",
     "help": "SSH connection timeout in seconds",
     "name": "ssh_conn_timeout",
     "required": "False",
     "secret": "False",
     "type": "Integer"},
    {"default": "",
     "deprecated_for_removal": "False",
     "help": "Filename of private key to use for SSH authentication",
     "name": "san_private_key",
     "required": "False",
     "secret": "False",
     "type": "String"},
    {"default": "1",
     "deprecated_for_removal": "False",
     "help": "The port that the NVMe target is listening on.",
     "name": "nvmet_port_id",
     "required": "False",
     "secret": "False",
     "type": "Port"},
    {"default": "",
     "deprecated_for_removal": "False",
     "help": "Certain ISCSI targets have predefined target names, SCST target "
             "driver uses this name.",
     "name": "scst_target_iqn_name",
     "required": "False",
     "secret": "False",
     "type": "String"},
    {"default": "iscsi",
     "deprecated_for_removal": "False",
     "help": "SCST target implementation can choose from multiple SCST target "
             "drivers.",
     "name": "scst_target_driver",
     "required": "False",
     "secret": "False",
     "type": "String"},
    {"default": "",
     "deprecated_for_removal": "False",
     "help": "The NVMe target remote configuration IP address.",
     "name": "spdk_rpc_ip",
     "required": "False",
     "secret": "False",
     "type": "String"},
    {"default": "8000",
     "deprecated_for_removal": "False",
     "help": "The NVMe target remote configuration port.",
     "name": "spdk_rpc_port",
     "required": "False",
     "secret": "False",
     "type": "Port"},
    {"default": "",
     "deprecated_for_removal": "False",
     "help": "The NVMe target remote configuration username.",
     "name": "spdk_rpc_username",
     "required": "False",
     "secret": "False",
     "type": "String"},
    {"default": "",
     "deprecated_for_removal": "False",
     "help": "The NVMe target remote configuration password.",
     "name": "spdk_rpc_password",
     "required": "False",
     "secret": "True",
     "type": "String"},
    {"default": "64",
     "deprecated_for_removal": "False",
     "help": "Queue depth for rdma transport.",
     "name": "spdk_max_queue_depth",
     "required": "False",
     "secret": "True",
     "type": "Integer(min=1, max=128)"},
)

MISSING_DRIVER_OPTIONS = {
    'QnapISCSI': ('target_ip_address', 'san_login', 'san_password',
                  'use_chap_auth', 'chap_username', 'chap_password',
                  'driver_ssl_cert_verify'),
    'XtremIOISCSI': ('san_ip', 'san_login', 'san_password',
                     'driver_ssl_cert_verify', 'driver_ssl_cert_path'),
    'XtremIOFC': ('san_ip', 'san_login', 'san_password',
                  'driver_ssl_cert_verify', 'driver_ssl_cert_path'),
    'LVMVolume': ('target_ip_address', 'target_helper', 'target_protocol',
                  'volume_clear', 'volume_clear_size', 'volume_dd_blocksize',
                  'target_prefix', 'volumes_dir', 'target_port',
                  'iscsi_secondary_ip_addresses',
                  'iscsi_write_cache', 'iscsi_target_flags',  # TGT
                  'iet_conf', 'iscsi_iotype',  # IET
                  'nvmet_port_id',  # NVMET
                  'scst_target_iqn_name', 'scst_target_driver',  # SCST
                  'spdk_rpc_ip', 'spdk_rpc_port', 'spdk_rpc_username',  # SPDK
                  'spdk_rpc_password', 'spdk_max_queue_depth',  # SPDK
                  ),
    'HPE3PARISCSI': ('san_ip', 'san_login', 'san_password', 'target_port',
                     'san_ssh_port', 'ssh_conn_timeout', 'san_private_key',
                     'target_ip_address'),
    'HPE3PARFC': ('san_ip', 'san_login', 'san_password', 'target_port',
                  'san_ssh_port', 'ssh_conn_timeout', 'san_private_key',
                  'target_ip_address'),
    'KaminarioISCSI': ('san_ip', 'san_login', 'san_password',
                       'volume_dd_blocksize'),
    'PowerMaxFC': ('san_ip', 'san_login', 'san_password',
                   'driver_ssl_cert_verify'),
    'PowerMaxISCSI': ('san_ip', 'san_login', 'san_password',
                      'driver_ssl_cert_verify', 'use_chap_auth',
                      'chap_username', 'chap_password'),
    'SynoISCSI': ('target_ip_address', 'target_protocol', 'driver_use_ssl',
                  'use_chap_auth', 'iscsi_secondary_ip_addresses',
                  'target_port', 'chap_username', 'chap_password',
                  'target_prefix'),
    'PureISCSI': ('san_ip', 'driver_ssl_cert_verify', 'driver_ssl_cert_path',
                  'use_chap_auth'),
    'PureFC': ('san_ip', 'driver_ssl_cert_verify', 'driver_ssl_cert_path',
               'use_chap_auth'),
    'SolidFire': ('san_ip', 'san_login', 'san_password',
                  'driver_ssl_cert_verify'),
}

IGNORE_OPTIONS = ['max_over_subscription_ratio',
                  'reserved_percentage',
                  'use_multipath_for_image_xfer',
                  'xtremio_volumes_per_glance_cache',
                  'auto_calc_max_oversubscription_ratio',
                  'replication_device',
                  'replication_connect_timeout',
                  ]

CONSOLE_VERSION = os.environ.get('CONSOLE_VERSION', '').strip() or '4.0'
CONSOLE_VERSION_TUPLE = list(map(int, (CONSOLE_VERSION + '.0').split('.')[:2]))
USE_GROUPS = CONSOLE_VERSION_TUPLE < [4, 4]

DROPDOWN_TEMPLATE = """        - description: The type of storage backend
          displayName: Driver
          path: config.envVars.X_CSI_BACKEND_CONFIG.driver
          x-descriptors:
            - 'urn:alm:descriptor:com.tectonic.ui:fieldGroup:DriverSettings'
${DROPDOWN_OPTIONS}"""
SAMPLE_TEMPLATE = collections.OrderedDict([
    ("apiVersion", "ember-csi.io/v1alpha1"),
    ("kind", "EmberStorageBackend"),
    ("metadata", {"name": "example"}),
    ("spec", {
        "config": collections.OrderedDict([
            ("envVars", collections.OrderedDict([
                ("X_CSI_PERSISTENCE_CONFIG", {"storage": "crd"}),
                # TODO: Operator to support X_CSI_ABORT_DUPLICATES as a boolean
                # and convert it to string form Ember-CSI
                # ("X_CSI_ABORT_DUPLICATES", False),
                ("X_CSI_DEBUG_MODE", ""),
                ("X_CSI_DEFAULT_MOUNT_FS", "ext4"),
                ("X_CSI_EMBER_CONFIG", {"plugin_name": "",
                                        "grpc_workers": 30,
                                        "slow_operations": True,
                                        "enable_probe": False,
                                        "disabled": [],
                                        "project_id": "ember_csi.io",
                                        "user_id": "ember_csi.io",
                                        "disable_logs": False,
                                        "debug": False,
                                        })])),
            ("sysFiles", {
                "name": "",
            })
        ])
    })
])


class Option(object):
    UI_PREFIX = "'urn:alm:descriptor:com.tectonic.ui:"

    def _set_default(self, option):
        self.default = '' if option['default'] == 'None' else option['default']

    def __init__(self, option):
        self.drivers = set()
        self.name = option['name']
        self.name_raw = self.name
        self.display_name = option['name'].replace('_', ' ').title()
        self.help = option['help'].replace('\n', '.').replace('"', "'").title()
        self.ignore = (option['deprecated_for_removal'] == 'True' or
                       option['name'] in IGNORE_OPTIONS)

        if option.get('secret') == 'True':
            self.form_type = self.UI_PREFIX + "password'"
            self._set_default(option)

        elif option['type'].startswith('String(choices='):
            start = option['type'].index('[') + 1
            end = option['type'].index(']')

            raw_choices = [c.strip()
                           for c in option['type'][start:end].split(',')]

            # Tell the operator if we have the None instance option (different
            # from 'none')
            if 'None' in raw_choices:
                # If we have multiple ways to say None, we leave the string
                # name
                if "'none'" in raw_choices:
                    raw_choices.remove('None')
                else:
                    self.name += TRANSFORM_POSSIBLE_NONE

            choices = ['' if c == 'None' else c.replace("'", '')
                       for c in raw_choices]
            self._set_default(option)
            prefix = self.UI_PREFIX + 'select:'
            self.form_type = prefix + ("'\n            - " + prefix).join(
                c.strip() for c in choices) + "'"

        # We don't have URI, IPAddress or Float types in the UI
        elif (option['type'].startswith('String')
              or option['type'] in ('URI', 'Float', 'IPAddress')):
            self.form_type = self.UI_PREFIX + "text'"
            self._set_default(option)
            if '(' in option['type']:
                self.help += ' ' + option['type'][option['type'].index('('):]
            # Tell the operator that this needs to be transformed
            if option['type'] == 'Float':
                self.name += TRANSFORM_STRING_FLOAT

        # booleanSwitch looks better, but it doesn't work for duplicated items
        elif option['type'] == 'Boolean':
            self.form_type = self.UI_PREFIX + "checkbox'"
            self.default = option['default'] == 'True'

        elif (option['type'].startswith('Port') or
              option['type'].startswith('Integer')):
            self.form_type = self.UI_PREFIX + "number'"
            self.default = ('' if option['default'] == 'None'
                            else int(float(option['default'])))

            # If we have min and max defined, add it to the help
            if '(' in option['type']:
                self.help += ' ' + option['type'][option['type'].index('('):]

        elif option['type'] == 'Dict of String':
            self.form_type = self.UI_PREFIX + "text'"
            # Tell the operator that this needs to be transformed
            self.name += TRANSFORM_DICT_OF_STRING

            default = option['default']
            if default in ('None', ''):
                self.default = ''
            else:
                default = json.loads(default.replace("'", '"'))
                self.default = ','.join('%s:%s' % i for i in default.items())

            self.help += ' [ie: k1:v1,k2:v2]'

        elif option['type'] in ('List of String', 'List of IPAddress'):
            self.form_type = self.UI_PREFIX + "text'"
            # Tell the operator that this needs to be transformed
            self.name += TRANSFORM_LIST_OF_STRINGS
            self.help += ' [ie: v1,v2]'
            # Convert the default (ie: ['Pool0']) into a CSV
            default = option['default']
            if default in ('None', ''):
                self.default = ''
            else:
                default_list = json.loads(default.replace("'", '"'))
                self.default = ','.join(default_list)

        else:
            raise Exception('Unkown type %s' % option['type'])

        # Optional depends on the driver, so don't indicate anything
        # if option['required'] == 'True':
        #     self.help += ' [Optional]'

    def add_driver(self, driver):
        self.drivers.add(driver)

    def option_name(self, driver=None):
        if driver:
            return "driver__%s__%s" % (driver, self.name)
        else:
            return self.name_raw


def include_driver(drivername):
    if EXCLUDE_DRIVERS is not None:
        for regex in EXCLUDE_DRIVERS:
            if re.match(regex, drivername, re.IGNORECASE):
                return False

    if INCLUDE_DRIVERS is None:
        return True

    for regex in INCLUDE_DRIVERS:
        if re.match(regex, drivername, re.IGNORECASE):
            return True

    return False


def render_option(option, driver, group_options=False):
    if option.ignore:
        return ''
    group = driver + "Options" if group_options else 'DriverSettings'

    result = """
        - description: {description}
          displayName: {display_name}
          path: config.envVars.X_CSI_BACKEND_CONFIG.{name}
          x-descriptors:
            - 'urn:alm:descriptor:com.tectonic.ui:fieldGroup:{group}'
            - {opt_type}
            - 'urn:alm:descriptor:com.tectonic.ui:fieldDependency:spec.config.envVars.X_CSI_BACKEND_CONFIG.driver:{driver}'""".format(
               name=option.option_name(driver),  # noqa
               display_name=option.display_name,
               description='"' + option.help + '"',
               opt_type=option.form_type,
               driver=driver,
               group=group)

    # The only way to present the groups folded is to set them as advanced
    if group_options:
        result += 12 * ' ' + "- 'urn:alm:descriptor:com.tectonic.ui:advanced'"
    # We don't indent here, it's the caller's responsibility
    return result


def additional_options():
    """Options that are generic for all drivers."""
    # TODO: Operator to support a JSON string for the X_CSI_PERSISTENCE_CONFIG
    # TODO: Operator to support X_CSI_ABORT_DUPLICATES as a boolean and convert
    #       it to string form Ember-CSI
    return """
        - description: Backend name (set by operator if empty)
          displayName: Name
          path: config.envVars.X_CSI_BACKEND_CONFIG.name
          x-descriptors:
            - 'urn:alm:descriptor:com.tectonic.ui:fieldGroup:AdvancedSettings'
            - 'urn:alm:descriptor:com.tectonic.ui:text'
            - 'urn:alm:descriptor:com.tectonic.ui:advanced'
        - description: The unique name of the plugin (set by operator if empty)
          displayName: Plugin name
          path: config.envVars.X_CSI_EMBER_CONFIG.plugin_name
          x-descriptors:
            - 'urn:alm:descriptor:com.tectonic.ui:fieldGroup:AdvancedSettings'
            - 'urn:alm:descriptor:com.tectonic.ui:text'
            - 'urn:alm:descriptor:com.tectonic.ui:advanced'
        - description: Number of gRPC workers for the CSI plugin
          displayName: gRPC workers
          path: config.envVars.X_CSI_EMBER_CONFIG.grpc_workers
          x-descriptors:
            - 'urn:alm:descriptor:com.tectonic.ui:fieldGroup:AdvancedSettings'
            - 'urn:alm:descriptor:com.tectonic.ui:number'
            - 'urn:alm:descriptor:com.tectonic.ui:advanced'
        - description: Allow keepalive pings when there are no gRPC calls
          displayName: Slow operations
          path: config.envVars.X_CSI_EMBER_CONFIG.slow_operations
          x-descriptors:
            - 'urn:alm:descriptor:com.tectonic.ui:fieldGroup:AdvancedSettings'
            - 'urn:alm:descriptor:com.tectonic.ui:checkbox'
            - 'urn:alm:descriptor:com.tectonic.ui:advanced'
        - description: Get stats from the storage backend when the CSI plugin is probed
          displayName: Probe backend
          path: config.envVars.X_CSI_EMBER_CONFIG.enable_probe
          x-descriptors:
            - 'urn:alm:descriptor:com.tectonic.ui:fieldGroup:AdvancedSettings'
            - 'urn:alm:descriptor:com.tectonic.ui:checkbox'
            - 'urn:alm:descriptor:com.tectonic.ui:advanced'
        - description: >
            List of features we want to disable on the plugin.
            Features that can be disabled are clone, snapshot, expand,
            expand_online. Must be a JSON list ie: ["clone", "expand_online"]'
          displayName: Disabled features
          path: config.envVars.X_CSI_EMBER_CONFIG.disabled__transform_csv
          x-descriptors:
            - 'urn:alm:descriptor:com.tectonic.ui:fieldGroup:AdvancedSettings'
            - 'urn:alm:descriptor:com.tectonic.ui:text'
            - 'urn:alm:descriptor:com.tectonic.ui:advanced'
        - description: Project ID to store in the persistence metadata backend
          displayName: Project ID
          path: config.envVars.X_CSI_EMBER_CONFIG.project_id
          x-descriptors:
            - 'urn:alm:descriptor:com.tectonic.ui:fieldGroup:AdvancedSettings'
            - 'urn:alm:descriptor:com.tectonic.ui:text'
            - 'urn:alm:descriptor:com.tectonic.ui:advanced'
        - description: User ID to store in the persistence metadata backend
          displayName: User ID
          path: config.envVars.X_CSI_EMBER_CONFIG.user_id
          x-descriptors:
            - 'urn:alm:descriptor:com.tectonic.ui:fieldGroup:AdvancedSettings'
            - 'urn:alm:descriptor:com.tectonic.ui:text'
            - 'urn:alm:descriptor:com.tectonic.ui:advanced'
        - description: Disable all the logs in the CSI plugins
          displayName: Quiet
          path: config.envVars.X_CSI_EMBER_CONFIG.disable_logs
          x-descriptors:
            - 'urn:alm:descriptor:com.tectonic.ui:fieldGroup:AdvancedSettings'
            - 'urn:alm:descriptor:com.tectonic.ui:checkbox'
            - 'urn:alm:descriptor:com.tectonic.ui:advanced'
        - description: Enabled debug log levels (quiet option must not be set)
          displayName: Debug logs
          path: config.envVars.X_CSI_EMBER_CONFIG.debug
          x-descriptors:
            - 'urn:alm:descriptor:com.tectonic.ui:fieldGroup:AdvancedSettings'
            - 'urn:alm:descriptor:com.tectonic.ui:checkbox'
            - 'urn:alm:descriptor:com.tectonic.ui:advanced'
        # - description: If we want to abort or queue (default) duplicated requests.
        #   displayName: Abort duplicates
        #   path: config.envVars.X_CSI_ABORT_DUPLICATES
        #   x-descriptors:
        #     - 'urn:alm:descriptor:com.tectonic.ui:fieldGroup:AdvancedSettings'
        #     - 'urn:alm:descriptor:com.tectonic.ui:checkbox'
        #     - 'urn:alm:descriptor:com.tectonic.ui:advanced'
        - description: Enable remote debugging with rpdb on calls
          displayName: Remote debugging
          path: config.envVars.X_CSI_DEBUG_MODE
          x-descriptors:
            - 'urn:alm:descriptor:com.tectonic.ui:fieldGroup:AdvancedSettings'
            - 'urn:alm:descriptor:com.tectonic.ui:select:'
            - 'urn:alm:descriptor:com.tectonic.ui:select:RPDB'
            - 'urn:alm:descriptor:com.tectonic.ui:advanced'
        - description: Default filesystem for mount type volumes when it is not specified
          displayName: Default filesystem
          path: config.envVars.X_CSI_DEFAULT_MOUNT_FS
          x-descriptors:
            - 'urn:alm:descriptor:com.tectonic.ui:fieldGroup:AdvancedSettings'
            - 'urn:alm:descriptor:com.tectonic.ui:select:btrfs'
            - 'urn:alm:descriptor:com.tectonic.ui:select:cramfs'
            - 'urn:alm:descriptor:com.tectonic.ui:select:ext2'
            - 'urn:alm:descriptor:com.tectonic.ui:select:ext3'
            - 'urn:alm:descriptor:com.tectonic.ui:select:ext4'
            - 'urn:alm:descriptor:com.tectonic.ui:select:minix'
            - 'urn:alm:descriptor:com.tectonic.ui:select:xfs'
            - 'urn:alm:descriptor:com.tectonic.ui:advanced'
        - description: Metadata persistence plugin selection and settings (must be valid JSON)
          displayName: Persistence
          path: config.envVars.X_CSI_PERSISTENCE_CONFIG
          x-descriptors:
            - 'urn:alm:descriptor:com.tectonic.ui:fieldGroup:AdvancedSettings'
            - 'urn:alm:descriptor:com.tectonic.ui:text'
            - 'urn:alm:descriptor:com.tectonic.ui:advanced'
        - description: Allow unsupported drivers to run
          displayName: Unsupported
          path: config.envVars.X_CSI_BACKEND_CONFIG.enable_unsupported_driver
          x-descriptors:
            - 'urn:alm:descriptor:com.tectonic.ui:fieldGroup:AdvancedSettings'
            - 'urn:alm:descriptor:com.tectonic.ui:checkbox'
            - 'urn:alm:descriptor:com.tectonic.ui:advanced'
        - description: sysFiles secrets
          displayName: sysFiles secrets
          path: config.sysFiles.name
          x-descriptors:
            - 'urn:alm:descriptor:com.tectonic.ui:fieldGroup:AdvancedSettings'
            - 'urn:alm:descriptor:io.kubernetes:Secret'
            - 'urn:alm:descriptor:com.tectonic.ui:advanced'
        - description: Use multipath if driver supports it
          displayName: Multipath
          path: config.X_CSI_BACKEND_CONFIG.multipath
          x-descriptors:
            - 'urn:alm:descriptor:com.tectonic.ui:fieldGroup:DriverSettings'
            - 'urn:alm:descriptor:com.tectonic.ui:checkbox'"""  # noqa


def _indent(text, spaces):
    if not spaces:
        return text

    padding = ' ' * spaces
    return padding + padding.join(line for line in text.splitlines(True))


def generate_driver_options(original_drivers, options):
    # For some reason the form presents it reversed
    names = sorted(original_drivers.keys(), reverse=True)
    option_element = 12 * ' ' + "- 'urn:alm:descriptor:com.tectonic.ui:select:"
    dropdown_options = "\n".join(option_element + key + "'" for key in names)

    dropdown = DROPDOWN_TEMPLATE.replace('${DROPDOWN_OPTIONS}',
                                         dropdown_options)

    rendered_opts = []
    # But the groups need to go in the right order
    for name in sorted(names):
        rendered_opts.extend(render_option(options[o], name, USE_GROUPS)
                             for o in original_drivers[name])

    driver_options = dropdown + ''.join(rendered_opts) + additional_options()
    return _indent(driver_options, ADDITIONAL_SPACES)


def generate_sample_options(name, option_names, options):
    cfg = collections.OrderedDict(name='', enable_unsupported_driver=False,
                                  driver=name)
    cfg.update((options[k].option_name(name), options[k].default)
               for k in option_names
               if not options[k].ignore)
    sample = copy.deepcopy(SAMPLE_TEMPLATE)
    sample['spec']['config']['envVars']['X_CSI_BACKEND_CONFIG'] = cfg
    return sample


def generate_sample_config(original_drivers, options):
    """Generate the whole alm-examples section."""
    # TODO: Ember-CSI will complain about a bunch of unknown configuration
    # options, since we'll pass ALL the existing configuration options.  Find a
    # way for the Operator to only pass relevant config options.
    # Generate individual examples with the defaults for each backend
    samples = [
        generate_sample_options(name, sorted(original_drivers[name]), options)
        for name in sorted(original_drivers.keys())
    ]
    used_options = sorted(set(itertools.chain.from_iterable(
        original_drivers[name] for name in original_drivers.keys())
    ))
    # We generate a sample with all the options and the first driver name
    defaults = samples[0]
    for sample in samples:
        s = sample['spec']['config']['envVars']['X_CSI_BACKEND_CONFIG']
        d = defaults['spec']['config']['envVars']['X_CSI_BACKEND_CONFIG']
        d.update(s)

    defaults['spec']['config']['envVars']['X_CSI_BACKEND_CONFIG']['driver'] = sorted(original_drivers.keys())[0]

    # The first examples is used for the defaults of the form.
    samples = [defaults]

    result = json.dumps(samples, indent=2)
    # We need to indent it
    examples = _indent(result, 6)
    res = '    alm-examples: |-\n' + examples
    # Remove trailing spaces introduced by JSON
    return '\n'.join(line.rstrip() for line in res.splitlines())


def generate_examples(original_drivers, options):
    """Generate one example config yaml per driver."""
    if os.path.exists("examples"):
        for name in sorted(original_drivers.keys()):
            sample = generate_sample_options(None, sorted(original_drivers[name]), options)
            sample['spec']['config']['envVars']['X_CSI_BACKEND_CONFIG']['driver'] = name
            fn = "examples/" + name + ".yaml"
            # Convert nested OrderedDicts to nested dicts
            cfg = json.loads(json.dumps(sample))
            yaml.safe_dump(cfg, file(fn,'w'), allow_unicode=True)


def add_missing_config_options(backends, options):
    for option_data in MISSING_OPTIONS:
        options.setdefault(option_data['name'], Option(option_data))

    for driver_name, option_names in MISSING_DRIVER_OPTIONS.items():
        if driver_name in backends:
            backends[driver_name].update(option_names)
            for option_name in option_names:
                options[option_name].add_driver(driver_name)


def parse_ember_data(data):
    result_drivers = {}
    result_options = {}

    for driver_name, v in data.items():
        driver_options = {o['name']: Option(o) for o in v['driver_options']}
        result_drivers[driver_name] = set(driver_options.keys())
        for option_name, option in driver_options.items():
            option = result_options.setdefault(option_name, option)
            option.add_driver(driver_name)
    return result_drivers, result_options


if __name__ == '__main__':
    data = json.load(sys.stdin)

    original_drivers, options = parse_ember_data(data)
    # Exclude drivers we don't support, like NFS
    backends = {k: v for k, v in original_drivers.items() if include_driver(k)}

    # Some drivers are not reporting all their options, fix the ones we know
    add_missing_config_options(backends, options)

    driver_options = generate_driver_options(backends, options)
    sample_config = generate_sample_config(backends, options)
    generate_examples(backends, options)

    template_name = 'template-dev.yaml' if DEVELOPMENT else 'template.yaml'
    with open(template_name, 'r') as f:
        template = f.read()
    result = template.replace('${DRIVER_OPTIONS}', driver_options)
    result = result.replace('${SAMPLE_CONFIG}', sample_config)
    print(result)

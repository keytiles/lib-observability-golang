
# release 1.2.1

New features:

Bug fixes:
 * removed 'custom' label from built in metric templates as not needed - and also updated README to show correct instantiation

Breaking changes:



# release 1.2.0

New features:
 * Added support to handle simple way our Metric standards - introducing
    * kt_observability_monitoring.GetExecCountTemplate()
    * kt_observability_monitoring.GetErrorCountTemplate()
    * kt_observability_monitoring.GetWarningCountTemplate()
    * kt_observability_monitoring.ProcessingTimeTemplate()
 * From now on kt_observability_monitoring also has GetGlobalLabels() and SetGlobalLabels() methods - just like it work in kt_logging

Bug fixes:

Breaking changes:


# release 1.1.0

New features:
 * Added kt_observability_logging.logging.go helper - this brings BuildDefaultGlobalLogLabels() method so log labels can setup similar to Metric labels

Bug fixes:

Breaking changes:


# release 1.0.2

Initial release

# release 1.0.1

Tried to fix issues but failed - retracted

# release 1.0.0

Original release but wrong repo name - retracted
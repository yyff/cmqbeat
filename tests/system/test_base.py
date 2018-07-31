from cmqbeat import BaseTest

import os


class Test(BaseTest):

    def test_base(self):
        """
        Basic test with exiting Cmqbeat normally
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*"
        )

        cmqbeat_proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("cmqbeat is running"))
        exit_code = cmqbeat_proc.kill_and_wait()
        assert exit_code == 0
